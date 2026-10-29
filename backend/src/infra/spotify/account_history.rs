//! Account listening history from Spotify's Web API, distinct from local listens.
use super::session::SpotifySession;
use anyhow::{bail, Context, Result};
use serde_json::Value;
use std::path::Path;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::OnceLock;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};
use tokio::sync::Mutex;

#[derive(Debug)]
struct HistoryFailure {
  reason: &'static str,
  retry_after: u64,
}
impl std::fmt::Display for HistoryFailure {
  fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
    f.write_str(self.reason)
  }
}
impl std::error::Error for HistoryFailure {}

fn now_seconds() -> u64 {
  SystemTime::now()
    .duration_since(UNIX_EPOCH)
    .unwrap_or_default()
    .as_secs()
}

fn cooldown_path() -> Option<std::path::PathBuf> {
  Some(crate::core::paths::app_state_dir()?.join("spotify-history-backoff.json"))
}

fn read_cooldown(path: &Path, now: u64) -> u64 {
  let read = || -> Option<u64> {
    if std::fs::metadata(path).ok()?.len() > 512 {
      return None;
    }
    let value: Value = serde_json::from_slice(&std::fs::read(path).ok()?).ok()?;
    Some(value["notBefore"].as_u64()?.saturating_sub(now))
  };
  read().unwrap_or(0)
}

fn save_cooldown(path: &Path, retry_after: u64) -> Result<()> {
  crate::core::paths::ensure_private_dir(path.parent().context("No history state directory")?)?;
  let bytes = serde_json::to_vec(
    &serde_json::json!({"notBefore":now_seconds().saturating_add(retry_after)}),
  )?;
  crate::core::auth::write_private_file_atomic(path, &bytes)
}

fn retry_after(headers: &reqwest::header::HeaderMap, now: u64) -> u64 {
  let value = headers
    .get(reqwest::header::RETRY_AFTER)
    .and_then(|v| v.to_str().ok());
  value
    .and_then(|v| {
      v.parse::<u64>().ok().or_else(|| {
        chrono::DateTime::parse_from_rfc2822(v)
          .ok()
          .map(|d| (d.timestamp().max(0) as u64).saturating_sub(now))
      })
    })
    .unwrap_or(300)
    .max(60)
}

/// Only the retry deadline is persisted; account history and bearer tokens stay in memory.
type CachedHistory = Option<(u64, Instant, Value)>;
fn cache() -> &'static Mutex<CachedHistory> {
  static CACHE: OnceLock<Mutex<CachedHistory>> = OnceLock::new();
  CACHE.get_or_init(|| Mutex::new(None))
}

fn cached_value(cache: &CachedHistory, generation: u64, now: Instant) -> Option<Value> {
  let (owner, until, value) = cache.as_ref()?;
  if *owner != generation || *until <= now {
    return None;
  }
  let mut value = value.clone();
  value["retryAfterSeconds"] = until.saturating_duration_since(now).as_secs().max(1).into();
  Some(value)
}

static GENERATION: AtomicU64 = AtomicU64::new(0);
pub fn clear() {
  GENERATION.fetch_add(1, Ordering::SeqCst);
  if let Ok(mut cache) = cache().try_lock() {
    *cache = None;
  }
}

pub async fn read() -> Value {
  let mut cache = cache().lock().await;
  let generation = GENERATION.load(Ordering::SeqCst);
  if let Some(value) = cached_value(&cache, generation, Instant::now()) {
    return value;
  }
  let cooldown = cooldown_path()
    .map(|path| read_cooldown(&path, now_seconds()))
    .unwrap_or(0);
  let result = if cooldown > 0 {
    Err(
      HistoryFailure {
        reason: "Spotify history rate-limited; waiting before retry",
        retry_after: cooldown,
      }
      .into(),
    )
  } else {
    tokio::time::timeout(Duration::from_secs(20), async {
      let session = super::session::current()
        .await
        .map_err(|_| anyhow::anyhow!("Spotify history connection unavailable"))?;
      recently_played(&session).await
    })
    .await
    .unwrap_or_else(|_| Err(anyhow::anyhow!("Spotify history request timed out")))
  };
  let (value, delay) = match result {
    Ok(mut value) => {
      value["retryAfterSeconds"] = 60.into();
      (value, 60)
    }
    Err(error) => {
      let delay = error
        .downcast_ref::<HistoryFailure>()
        .map(|e| e.retry_after)
        .unwrap_or(60);
      (
        serde_json::json!({"items":[],"err":error.to_string(),"retryAfterSeconds":delay}),
        delay,
      )
    }
  };
  if generation != GENERATION.load(Ordering::SeqCst) {
    *cache = None;
    return serde_json::json!({"items":[],"err":"History request cancelled","retryAfterSeconds":60});
  }
  let until = Instant::now()
    .checked_add(Duration::from_secs(delay))
    .unwrap_or_else(|| Instant::now() + Duration::from_secs(31536000));
  *cache = Some((generation, until, value.clone()));
  value
}

pub async fn recently_played(session: &SpotifySession) -> Result<Value> {
  let token = session
    .session()
    .login5()
    .auth_token()
    .await
    .map_err(|_| anyhow::anyhow!("Spotify history authorization unavailable"))?;
  let client = reqwest::Client::builder()
    .timeout(Duration::from_secs(12))
    .redirect(reqwest::redirect::Policy::none())
    .build()?;
  request_history(
    &client,
    "https://api.spotify.com/v1/me/player/recently-played",
    &token.access_token,
    cooldown_path().as_deref(),
  )
  .await
}

async fn request_history(
  client: &reqwest::Client,
  url: &str,
  token: &str,
  cooldown: Option<&Path>,
) -> Result<Value> {
  let mut response = client
    .get(url)
    .query(&[("limit", "50")])
    .bearer_auth(token)
    .send()
    .await
    .context("Spotify history request failed")?;
  if !response.status().is_success() {
    let (reason, delay) = match response.status().as_u16() {
      429 => {
        let wait = retry_after(response.headers(), now_seconds());
        if let Some(path) = cooldown {
          let _ = save_cooldown(path, wait);
        }
        ("Spotify history rate-limited; waiting before retry", wait)
      }
      401 => ("Spotify history login expired", 300),
      403 => ("Spotify denied access to listening history", 300),
      _ => ("Spotify history service unavailable", 60),
    };
    return Err(
      HistoryFailure {
        reason,
        retry_after: delay,
      }
      .into(),
    );
  }
  let mut bytes = Vec::new();
  while let Some(chunk) = response.chunk().await? {
    if bytes.len() + chunk.len() > 2 * 1024 * 1024 {
      bail!("Spotify history response exceeds 2 MiB");
    }
    bytes.extend_from_slice(&chunk);
  }
  let value: Value = serde_json::from_slice(&bytes)?;
  if !value["items"].is_array() {
    bail!("Spotify history response is missing items");
  }
  Ok(value)
}

#[cfg(test)]
mod tests {
  use super::*;
  use std::io::{Read, Write};
  #[test]
  fn old_account_cache_is_not_reused_after_session_generation_changes() {
    let now = Instant::now();
    let entry = Some((
      7,
      now + Duration::from_secs(60),
      serde_json::json!({"items":["old-account-play"]}),
    ));
    assert!(cached_value(&entry, 7, now).is_some());
    assert!(cached_value(&entry, 8, now).is_none());
    assert!(cached_value(&entry, 7, now + Duration::from_secs(61)).is_none());
  }
  fn serve(headers: &str, body: Vec<u8>) -> (String, std::thread::JoinHandle<String>) {
    let listener = std::net::TcpListener::bind("127.0.0.1:0").unwrap();
    let url = format!("http://{}", listener.local_addr().unwrap());
    let headers = format!(
      "HTTP/1.1 {headers}\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
      body.len()
    );
    let thread = std::thread::spawn(move || {
      let (mut socket, _) = listener.accept().unwrap();
      socket
        .set_read_timeout(Some(Duration::from_secs(3)))
        .unwrap();
      let mut request = [0; 4096];
      let n = socket.read(&mut request).unwrap();
      let _ = socket.write_all(headers.as_bytes());
      let _ = socket.write_all(&body);
      String::from_utf8_lossy(&request[..n]).into_owned()
    });
    (url, thread)
  }
  fn client() -> reqwest::Client {
    reqwest::Client::builder()
      .timeout(Duration::from_secs(3))
      .redirect(reqwest::redirect::Policy::none())
      .build()
      .unwrap()
  }
  #[tokio::test]
  async fn account_history_uses_bearer_auth_and_keeps_spotify_order() {
    let (url,thread)=serve("200 OK",br#"{"items":[{"played_at":"newer","track":{"id":"first"}},{"played_at":"older","track":{"id":"second"}}]}"#.to_vec());
    let value = request_history(&client(), &url, "fixture-token", None)
      .await
      .unwrap();
    assert_eq!(value["items"][0]["track"]["id"], "first");
    let request = thread.join().unwrap().to_lowercase();
    assert!(request.contains("authorization: bearer fixture-token"));
    assert!(request.contains("limit=50"));
  }
  #[tokio::test]
  async fn rate_limit_deadline_survives_restart_without_saving_history_or_token() {
    let dir = tempfile::tempdir().unwrap();
    let path = dir.path().join("backoff.json");
    let (url, thread) = serve(
      "429 Too Many Requests\r\nRetry-After: 7200",
      b"private error body".to_vec(),
    );
    let error = request_history(&client(), &url, "fixture-token", Some(&path))
      .await
      .unwrap_err();
    thread.join().unwrap();
    assert_eq!(
      error.downcast_ref::<HistoryFailure>().unwrap().retry_after,
      7200
    );
    assert!(read_cooldown(&path, now_seconds()) >= 7198);
    let saved = std::fs::read_to_string(&path).unwrap();
    assert!(!saved.contains("fixture-token") && !saved.contains("private"));
    assert_eq!(
      serde_json::from_str::<Value>(&saved)
        .unwrap()
        .as_object()
        .unwrap()
        .len(),
      1
    );
  }
  #[tokio::test]
  async fn redirects_and_malformed_or_oversized_history_are_rejected() {
    for (status, body) in [
      ("302 Found\r\nLocation: http://127.0.0.1:1", Vec::new()),
      ("200 OK", b"{}".to_vec()),
      ("200 OK", vec![b'x'; 2 * 1024 * 1024 + 1]),
    ] {
      let (url, thread) = serve(status, body);
      assert!(request_history(&client(), &url, "fixture-token", None)
        .await
        .is_err());
      thread.join().unwrap();
    }
  }
  #[test]
  fn retry_after_accepts_seconds_and_http_dates() {
    let mut headers = reqwest::header::HeaderMap::new();
    headers.insert(reqwest::header::RETRY_AFTER, "7200".parse().unwrap());
    assert_eq!(retry_after(&headers, 0), 7200);
    headers.insert(
      reqwest::header::RETRY_AFTER,
      "Thu, 01 Jan 1970 02:00:00 GMT".parse().unwrap(),
    );
    assert_eq!(retry_after(&headers, 0), 7200);
    headers.insert(reqwest::header::RETRY_AFTER, "invalid".parse().unwrap());
    assert_eq!(retry_after(&headers, 0), 300);
  }
  #[tokio::test]
  #[ignore = "requires the user's existing cached Spotify login"]
  async fn real_account_history_is_available() {
    super::super::session::ALLOW_IN_TESTS.store(true, std::sync::atomic::Ordering::Relaxed);
    let value = super::read().await;
    assert!(
      value["err"].is_null(),
      "{}; retry after {} seconds",
      value["err"],
      value["retryAfterSeconds"]
    );
    println!(
      "Spotify account history records: {}",
      value["items"].as_array().unwrap().len()
    );
  }
}
