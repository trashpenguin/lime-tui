//! Official Soloist playback for accounts that cannot stream through librespot.
//! The executable is installed by the user from Spotify, never redistributed.
use super::*;
use futures::{SinkExt, StreamExt};
use std::{
  path::PathBuf,
  process::Stdio,
  sync::atomic::{AtomicBool, Ordering},
};
use tokio::{process::Child, sync::Notify};
use tokio_tungstenite::tungstenite::{protocol::WebSocketConfig, Message};

pub(super) struct Player {
  child: Mutex<Option<Child>>,
  commands: mpsc::Sender<Value>,
  stop: Notify,
  pub connected: AtomicBool,
  pub paired: AtomicBool,
  #[cfg(windows)]
  _job: Option<job::Job>,
}
impl Player {
  pub async fn shutdown(&self) {
    self.stop.notify_one();
    self.connected.store(false, Ordering::Release);
    if let Some(mut child) = self.child.lock().await.take() {
      let _ = child.kill().await;
    }
  }
  pub fn send(&self, command: Value) -> Result<()> {
    if !self.connected.load(Ordering::Acquire) {
      bail!("Soloist connection ended. Press R to reconnect.");
    }
    if !self.paired.load(Ordering::Acquire) {
      bail!("Open Spotify on the same network and select Lime Soloist in the device menu to pair.");
    }
    self
      .commands
      .try_send(command)
      .map_err(|_| anyhow!("Soloist is busy; try again shortly"))
  }
}

pub(super) async fn start(backend: &Arc<Backend>, params: &Value) -> Result<()> {
  let _guard = backend.commands.lock().await;
  let path = PathBuf::from(text(params, "path"));
  let key = text(params, "key");
  if !path.is_absolute() || !path.is_file() || key.is_empty() || key.len() > 4096 {
    bail!("Run Setup Soloist.cmd to select the official executable and enter your own key.");
  }
  if let Some(old) = backend.soloist.lock().await.take() {
    old.shutdown().await;
  }
  if let Some(old) = backend.player.lock().await.take() {
    old.shutdown();
  }
  backend.queue.lock().await.freeze();
  let data = crate::core::paths::app_state_dir()
    .context("No state directory")?
    .join("lime-soloist");
  let cache = crate::core::paths::app_cache_dir()
    .context("No cache directory")?
    .join("lime-soloist");
  crate::core::paths::ensure_private_dir(&data)?;
  crate::core::paths::ensure_private_dir(&cache)?;
  let volume = backend.queue.lock().await.volume.to_string();
  // The official CLI accepts the key on argv. Never log the command or its output.
  let mut command = tokio::process::Command::new(path);
  command
    .args([
      "-n",
      "Lime Soloist",
      "-k",
      key,
      "-w",
      "127.0.0.1:0",
      "-i",
      &volume,
      "-z",
      "100",
    ])
    .arg("-D")
    .arg(&data)
    .arg("-C")
    .arg(&cache)
    .stdin(Stdio::null())
    .stdout(Stdio::null())
    .stderr(Stdio::null())
    .kill_on_drop(true);
  #[cfg(windows)]
  command.creation_flags(0x08000000);
  let mut child = command
    .spawn()
    .map_err(|_| anyhow!("Could not start Soloist. Check its path and Windows x64 build."))?;
  #[cfg(windows)]
  let job = job::Job::contain(&child)
    .ok_or_else(|| anyhow!("Could not contain the Soloist child process"))?;
  let pid = child.id().context("Soloist did not start")?;
  let deadline = Instant::now() + Duration::from_secs(30);
  let socket = loop {
    if child.try_wait()?.is_some() {
      bail!(
        "Soloist exited. Check your API key, build expiry, and whether another Lime instance is running."
      );
    }
    if Instant::now() >= deadline {
      bail!("Soloist control API did not start. Check its build and API key.");
    }
    if let Some(port) = owned_port(&data, pid) {
      let config = WebSocketConfig::default()
        .max_message_size(Some(1024 * 1024))
        .max_frame_size(Some(1024 * 1024));
      if let Ok(Ok((socket, _))) = tokio::time::timeout(
        Duration::from_secs(2),
        tokio_tungstenite::connect_async_with_config(
          format!("ws://127.0.0.1:{port}"),
          Some(config),
          true,
        ),
      )
      .await
      {
        break socket;
      }
    }
    tokio::time::sleep(Duration::from_millis(150)).await;
  };
  let (tx, mut rx) = mpsc::channel::<Value>(32);
  let player = Arc::new(Player {
    child: Mutex::new(Some(child)),
    commands: tx,
    stop: Notify::new(),
    connected: AtomicBool::new(true),
    paired: AtomicBool::new(false),
    #[cfg(windows)]
    _job: Some(job),
  });
  *backend.soloist.lock().await = Some(player.clone());
  let weak_backend = Arc::downgrade(backend);
  let weak_player = Arc::downgrade(&player);
  tokio::spawn(async move {
    let (mut sink, mut stream) = socket.split();
    loop {
      let Some(player) = weak_player.upgrade() else {
        break;
      };
      let event = tokio::select! {
        _ = player.stop.notified() => break,
        command = rx.recv() => {
          let Some(command) = command else { break; };
          if sink.send(Message::Text(command.to_string().into())).await.is_err() { break; }
          continue;
        },
        event = stream.next() => event,
      };
      let Some(backend) = weak_backend.upgrade() else {
        break;
      };
      match event {
        Some(Ok(Message::Text(text))) => {
          if let Ok(value) = serde_json::from_str::<Value>(&text) {
            apply(&backend, &player, &value).await;
          }
        }
        Some(Ok(Message::Ping(bytes))) => {
          if sink.send(Message::Pong(bytes)).await.is_err() {
            break;
          }
        }
        Some(Ok(Message::Close(_))) | Some(Err(_)) | None => break,
        _ => {}
      }
    }
    if let (Some(backend), Some(player)) = (weak_backend.upgrade(), weak_player.upgrade()) {
      player.connected.store(false, Ordering::Release);
      if current(&backend, &player).await {
        let mut q = backend.queue.lock().await;
        q.freeze();
        q.error =
          "Soloist disconnected. Press R to reconnect; update Soloist if its build has expired."
            .into();
      }
    }
  });
  Ok(())
}

pub(super) fn clear_session() -> Result<()> {
  for root in [
    crate::core::paths::app_state_dir(),
    crate::core::paths::app_cache_dir(),
  ]
  .into_iter()
  .flatten()
  {
    let path = root.join("lime-soloist");
    match std::fs::symlink_metadata(&path) {
      Ok(meta) if meta.file_type().is_symlink() || !meta.is_dir() => {
        bail!("Soloist session path is not a private directory")
      }
      Ok(_) => {
        std::fs::remove_dir_all(path).context("Could not clear the local Soloist session")?;
      }
      Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
      Err(_) => bail!("Could not inspect the local Soloist session"),
    }
  }
  Ok(())
}
fn owned_port(data: &std::path::Path, pid: u32) -> Option<u16> {
  let read = |name: &str| {
    let path = data.join(name);
    (std::fs::metadata(&path).ok()?.len() <= 128).then_some(())?;
    std::fs::read_to_string(path).ok()
  };
  if read("soloist.pid")?.trim().parse::<u32>().ok()? != pid {
    return None;
  }
  read("ws.port")?
    .trim()
    .parse::<u16>()
    .ok()
    .filter(|port| *port != 0)
}
async fn current(backend: &Backend, player: &Arc<Player>) -> bool {
  backend
    .soloist
    .lock()
    .await
    .as_ref()
    .is_some_and(|p| Arc::ptr_eq(p, player))
}
pub(super) async fn load(backend: &Backend, player: &Arc<Player>, index: usize) -> Result<()> {
  let mut q = backend.queue.lock().await;
  if index >= q.tracks.len() {
    bail!("Queue position is out of range");
  }
  let uri = format!("spotify:track:{}", q.tracks[index].id);
  player.send(command("activate"))?;
  player.send(json!({"type":"command", "command":"play", "uri":uri}))?;
  q.freeze();
  q.index = index;
  q.position = 0;
  q.loading = true;
  q.play_intent = true;
  q.error.clear();
  q.generation = q.generation.wrapping_add(1);
  q.request_id = None;
  q.soloist_track_confirmed = false;
  q.trace.push("load_sent", None, Some(true), 0);
  Ok(())
}
pub(super) fn command(name: &str) -> Value {
  json!({"type":"command", "command":name})
}

async fn apply(backend: &Arc<Backend>, player: &Arc<Player>, value: &Value) {
  if !current(backend, player).await {
    return;
  }
  let kind = text(value, "type");
  if kind == "auth_state" {
    let paired = value["logged_in"].as_bool().unwrap_or(false);
    player.paired.store(paired, Ordering::Release);
    let mut q = backend.queue.lock().await;
    if !paired {
      q.freeze();
      q.error =
        "Pair Soloist: open Spotify on this network and select Lime Soloist in its device menu."
          .into();
    } else {
      q.error.clear();
    }
    return;
  }
  // Only playback events confirm playback. command_result means dispatched, not playing.
  if kind == "command_result" {
    return;
  }
  if kind == "error" {
    let mut q = backend.queue.lock().await;
    q.freeze();
    q.error =
      "Soloist rejected the playback command. Check pairing, then press R to reconnect.".into();
    return;
  }
  let mut q = backend.queue.lock().await;
  match kind {
    "playback_state" | "track_changed" => {
      let uri = text(&value["item"], "uri");
      if !uri.is_empty() && q.current_uri().as_deref() != Some(uri) {
        if q.loading {
          return;
        } // Older track event while a new Play is in flight.
        q.freeze();
        q.error =
          "Playback changed in Spotify. Select a Lime queue track to resume control.".into();
        return;
      }
      if !uri.is_empty() && q.current_uri().as_deref() == Some(uri) {
        q.soloist_track_confirmed = true;
      }
      if kind == "playback_state" {
        if value["is_active"].as_bool() == Some(false) {
          q.freeze();
          return;
        }
        let position = anchor_position(&value["position"]).unwrap_or(q.position);
        apply_status(&mut q, text(value, "status"), position);
        if let Some(volume) = value["volume"].as_u64() {
          q.volume = volume.min(100) as u8;
        }
      }
    }
    "playback_changed" => {
      let position = q.position_ms();
      let status = text(value, "status");
      if q.loading && !q.soloist_track_confirmed {
        return;
      }
      if status == "idle" && q.play_intent && q.playing {
        let duration = q.tracks.get(q.index).map(|t| t.dur_ms).unwrap_or(0);
        if duration > 0 && position.saturating_add(5000) >= duration {
          let next = q.next_index(true);
          let generation = q.generation;
          q.freeze();
          q.play_intent = next.is_some();
          drop(q);
          if let Some(index) = next {
            let _guard = backend.commands.lock().await;
            let valid = {
              let q = backend.queue.lock().await;
              q.generation == generation && q.play_intent
            };
            if valid && current(backend, player).await {
              if let Err(error) = load(backend, player, index).await {
                backend.playback_error(error).await;
              }
            }
          }
          return;
        }
      }
      apply_status(&mut q, status, position);
    }
    "position_sync" => {
      if let Some(position) = anchor_position(&value["position"]) {
        let playing = q.playing;
        q.reset_clock(position, playing);
      }
    }
    "volume_changed" => {
      if let Some(volume) = value["volume"].as_u64() {
        q.volume = volume.min(100) as u8;
      }
    }
    "device_changed" if value["is_active"].as_bool() == Some(false) => {
      q.freeze();
      q.trace.push("handoff", None, Some(true), 0);
    }
    _ => {}
  }
}
fn anchor_position(value: &Value) -> Option<u64> {
  let position = value["position_ms"].as_u64()?;
  let at = value["timestamp_ms"].as_i64()?;
  let speed = value["speed"].as_f64().unwrap_or(1.0).clamp(0.0, 4.0);
  Some(
    position
      .saturating_add(((chrono::Utc::now().timestamp_millis() - at).max(0) as f64 * speed) as u64),
  )
}
fn apply_status(q: &mut Queue, status: &str, position: u64) {
  match status {
    "playing" if q.play_intent => {
      q.reset_clock(position, true);
      q.play_intent = true;
      q.error.clear();
      q.trace.push("playing", None, Some(true), position);
    }
    "buffering" => {
      q.reset_clock(position, false);
      q.loading = true;
    }
    "paused" | "idle" => {
      q.freeze();
      q.position = position;
    }
    _ => {}
  }
}

#[cfg(windows)]
mod job {
  use windows_sys::Win32::{
    Foundation::{CloseHandle, HANDLE},
    System::JobObjects::*,
  };
  pub(super) struct Job(HANDLE);
  // SAFETY: this owned kernel handle may be sent/shared and is closed once by Drop.
  unsafe impl Send for Job {}
  unsafe impl Sync for Job {}
  impl Job {
    pub(super) fn contain(child: &tokio::process::Child) -> Option<Self> {
      let process = child.raw_handle()?;
      // SAFETY: a fresh Job contains only the owned Soloist child.
      unsafe {
        let handle = CreateJobObjectW(std::ptr::null(), std::ptr::null());
        if handle.is_null() {
          return None;
        }
        let job = Self(handle);
        let mut info: JOBOBJECT_EXTENDED_LIMIT_INFORMATION = std::mem::zeroed();
        info.BasicLimitInformation.LimitFlags = JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;
        let set = SetInformationJobObject(
          handle,
          JobObjectExtendedLimitInformation,
          std::ptr::from_ref(&info).cast(),
          std::mem::size_of_val(&info) as u32,
        ) != 0;
        (set && AssignProcessToJobObject(handle, process as HANDLE) != 0).then_some(job)
      }
    }
  }
  impl Drop for Job {
    fn drop(&mut self) {
      unsafe {
        CloseHandle(self.0);
      }
    }
  }
}

#[cfg(test)]
mod tests {
  use super::*;
  async fn fixture() -> (Arc<Backend>, Arc<Player>, mpsc::Receiver<Value>) {
    let backend = Arc::new(Backend::new());
    *backend.queue.lock().await = Queue {
      tracks: (0..2)
        .map(|i| Track {
          id: format!("{i:022}"),
          dur_ms: 180000,
          ..Track::default()
        })
        .collect(),
      ..Queue::default()
    };
    let (tx, rx) = mpsc::channel(32);
    let player = Arc::new(Player {
      child: Mutex::new(None),
      commands: tx,
      stop: Notify::new(),
      connected: AtomicBool::new(true),
      paired: AtomicBool::new(true),
      #[cfg(windows)]
      _job: None,
    });
    *backend.soloist.lock().await = Some(player.clone());
    (backend, player, rx)
  }
  #[tokio::test]
  async fn dispatch_waits_for_matching_track_and_real_playback() {
    let (backend, player, mut rx) = fixture().await;
    load(&backend, &player, 0).await.unwrap();
    assert_eq!(rx.recv().await.unwrap()["command"], "activate");
    let sent = rx.recv().await.unwrap();
    assert_eq!(sent["command"], "play");
    apply(
      &backend,
      &player,
      &json!({"type":"command_result","success":true}),
    )
    .await;
    apply(
      &backend,
      &player,
      &json!({"type":"playback_changed","status":"playing"}),
    )
    .await;
    assert!(!backend.queue.lock().await.playing);
    apply(
      &backend,
      &player,
      &json!({"type":"track_changed","item":{"uri":sent["uri"]}}),
    )
    .await;
    apply(
      &backend,
      &player,
      &json!({"type":"playback_changed","status":"playing"}),
    )
    .await;
    assert!(backend.queue.lock().await.playing);
    assert!(backend.player.lock().await.is_none());
  }
  #[tokio::test]
  async fn retirement_pause_and_handoff_do_not_reclaim_playback() {
    let (backend, player, mut rx) = fixture().await;
    load(&backend, &player, 0).await.unwrap();
    rx.recv().await.unwrap();
    rx.recv().await.unwrap();
    backend.queue.lock().await.freeze();
    apply(&backend,&player,&json!({"type":"playback_state","is_active":true,"status":"playing","item":{"uri":format!("spotify:track:{:022}",0)}})).await;
    assert!(!backend.queue.lock().await.playing);
    apply(
      &backend,
      &player,
      &json!({"type":"device_changed","is_active":false}),
    )
    .await;
    assert!(rx.try_recv().is_err());
    *backend.soloist.lock().await = None;
    apply(
      &backend,
      &player,
      &json!({"type":"error","message":"private-fixture"}),
    )
    .await;
    assert!(backend.queue.lock().await.error.is_empty());
  }
  #[tokio::test]
  async fn real_end_advances_once_and_errors_never_copy_remote_text() {
    let (backend, player, mut rx) = fixture().await;
    {
      let mut q = backend.queue.lock().await;
      q.position = 179500;
      q.playing = true;
      q.play_intent = true;
    }
    apply(
      &backend,
      &player,
      &json!({"type":"playback_changed","status":"idle"}),
    )
    .await;
    assert_eq!(rx.recv().await.unwrap()["command"], "activate");
    let sent = rx.recv().await.unwrap();
    assert_eq!(sent["uri"], format!("spotify:track:{:022}", 1));
    apply(
      &backend,
      &player,
      &json!({"type":"playback_changed","status":"idle"}),
    )
    .await;
    assert!(rx.try_recv().is_err());
    apply(
      &backend,
      &player,
      &json!({"type":"error","message":"private-fixture"}),
    )
    .await;
    let q = backend.queue.lock().await;
    assert!(!q.playing);
    assert!(!q.loading);
    assert!(!q.error.contains("private-fixture"));
  }
  #[test]
  fn a_dispatched_command_is_not_playback_confirmation() {
    let mut q = Queue {
      loading: true,
      play_intent: true,
      ..Queue::default()
    };
    apply_status(&mut q, "dispatched", 0);
    assert!(!q.playing);
    assert!(q.loading);
    apply_status(&mut q, "playing", 100);
    assert!(q.playing);
    assert!(!q.loading);
  }
  #[test]
  fn endpoint_files_must_belong_to_the_child() {
    let dir = tempfile::tempdir().unwrap();
    std::fs::write(dir.path().join("soloist.pid"), "12").unwrap();
    std::fs::write(dir.path().join("ws.port"), "54321").unwrap();
    assert_eq!(owned_port(dir.path(), 13), None);
    assert_eq!(owned_port(dir.path(), 12), Some(54321));
  }
}
