//! A small JSON-lines backend for the Pomme Bubble Tea frontend.
//! Spotify transport/auth/catalog are reused from spotatui; this frontend owns
//! its bounded queue and does not start spotatui's terminal, artwork, or plugins.
use crate::infra::player::{PlayerEvent, StreamingConfig, StreamingPlayer};
use crate::infra::spotify::{self, credentials, session};
use anyhow::{anyhow, bail, Context, Result};
use reqwest::Method;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::{
  io::Write,
  sync::Arc,
  time::{Duration, Instant},
};
use tokio::sync::{mpsc, Mutex, Semaphore};

#[path = "pomme_soloist.rs"]
mod soloist;

const MAX_QUEUE: usize = 1000;
const MAX_LINE: usize = 64 * 1024;

#[derive(Debug, PartialEq)]
enum TrackEnd {
  Complete,
  RetryStart,
  Failed,
}
fn classify_track_end(
  position: u64,
  duration: u64,
  wants_playback: bool,
  retried: bool,
) -> TrackEnd {
  if duration == 0 || position.saturating_add(5000) >= duration {
    return TrackEnd::Complete;
  }
  if wants_playback && position <= 8000 && !retried {
    TrackEnd::RetryStart
  } else {
    TrackEnd::Failed
  }
}

fn load_failure_message(stage: librespot_playback::player::LoadStage) -> &'static str {
  use librespot_playback::player::LoadStage;
  match stage {
    LoadStage::Metadata => {
      "Spotify track metadata request failed. Press R to reconnect, then try again."
    }
    LoadStage::Availability => {
      "Spotify marked this track unavailable for this account. Try another track."
    }
    LoadStage::Format => {
      "Spotify returned no supported audio format for this track. Try another track."
    }
    LoadStage::Download => {
      "Spotify audio download failed. Check the connection, then press R to reconnect."
    }
    LoadStage::AudioKey => {
      "Spotify audio-key request failed. Press R to reconnect; if it persists, save an F12 report."
    }
    LoadStage::Decode => "The Spotify audio stream could not be decoded. Save an F12 report.",
    LoadStage::Seek => {
      "The Spotify stream could not start at the requested position. Try the track again."
    }
    LoadStage::LocalFile => "The local audio file could not be opened or decoded.",
    LoadStage::Internal => "The Spotify track loader stopped unexpectedly. Press R to reconnect.",
  }
}

#[derive(Default)]
struct PlaybackTrace {
  started: Option<Instant>,
  entries: std::collections::VecDeque<Value>,
}
impl PlaybackTrace {
  fn annotate_load_failure(&mut self, failure: librespot_playback::player::LoadFailure) {
    if let Some(entry) = self
      .entries
      .back_mut()
      .filter(|entry| entry["event"] == "unavailable")
    {
      entry["failureStage"] = json!(failure.stage.code());
      entry["failureKind"] = json!(failure.kind.map(|kind| kind as i32));
    }
  }
  fn push(
    &mut self,
    event: &'static str,
    request: Option<u64>,
    current: Option<bool>,
    position: u64,
  ) {
    let at = self
      .started
      .get_or_insert_with(Instant::now)
      .elapsed()
      .as_millis() as u64;
    if self.entries.len() == 64 {
      self.entries.pop_front();
    }
    self.entries.push_back(json!({"atMs":at,"event":event,"request":request,"matchesCurrent":current,"positionMs":position}));
  }
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct Track {
  id: String,
  title: String,
  artist: String,
  album: String,
  kind: String,
  dur_ms: u64,
  art: String,
}

#[derive(Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
struct Queue {
  tracks: Vec<Track>,
  index: usize,
  position: u64,
  volume: u8,
  shuffle: bool,
  repeat: u8,
  #[serde(skip)]
  playing: bool,
  #[serde(skip)]
  loading: bool,
  #[serde(skip)]
  clock: Option<Instant>,
  #[serde(skip)]
  error: String,
  #[serde(skip)]
  trace: PlaybackTrace,
  #[serde(skip)]
  soloist_track_confirmed: bool,
  #[serde(skip)]
  request_id: Option<u64>,
  #[serde(skip)]
  observed_request: Option<u64>,
  #[serde(skip)]
  connect_queue: bool,
  #[serde(skip)]
  generation: u64,
  #[serde(skip)]
  retry_used: bool,
  #[serde(skip)]
  retry_pending: bool,
  #[serde(skip)]
  play_intent: bool,
}

impl Queue {
  // Decoder request IDs increase for both local loads and Spotify Connect loads.
  // A newer request from Connect is authoritative; older buffered events are not.
  fn observe_connect_track(&mut self, uri: &str, request: u64) -> bool {
    let Some(id) = uri.strip_prefix("spotify:track:") else {
      return false;
    };
    if id.len() != 22 || !id.bytes().all(|b| b.is_ascii_alphanumeric()) {
      return false;
    }
    if let Some(active) = self.request_id {
      if request < active {
        return false;
      }
      if request == active {
        return self.current_uri().as_deref() == Some(uri);
      }
    }
    self.freeze();
    self.generation = self.generation.wrapping_add(1);
    self.retry_used = false;
    self.error.clear();
    self.position = 0;
    self.request_id = Some(request);
    self.connect_queue = true;
    self.play_intent = true;
    self.index = if let Some(index) = self.tracks.iter().position(|t| t.id == id) {
      index
    } else {
      // Keep the existing queue bounded, including repeated phone selections.
      if self.tracks.len() == MAX_QUEUE {
        self.tracks.remove(0);
      }
      self.tracks.push(Track {
        id: id.into(),
        title: "Loading Spotify track…".into(),
        kind: "song".into(),
        ..Track::default()
      });
      self.tracks.len() - 1
    };
    true
  }

  fn position_ms(&self) -> u64 {
    let elapsed = if self.playing {
      self
        .clock
        .map(|t| t.elapsed().as_millis() as u64)
        .unwrap_or(0)
    } else {
      0
    };
    let duration = self.tracks.get(self.index).map(|t| t.dur_ms).unwrap_or(0);
    (self.position + elapsed).min(duration)
  }
  fn freeze(&mut self) {
    self.position = self.position_ms();
    self.clock = None;
    self.playing = false;
    self.loading = false;
    self.retry_pending = false;
    self.play_intent = false;
  }
  fn owns_event(&self, uri: &str, request: u64) -> bool {
    self.current_uri().as_deref() == Some(uri)
      && self.request_id.is_none_or(|active| active == request)
  }
  fn reset_clock(&mut self, position: u64, playing: bool) {
    self.position = position;
    self.playing = playing;
    self.clock = playing.then(Instant::now);
    self.loading = false;
  }
  fn current_uri(&self) -> Option<String> {
    self
      .tracks
      .get(self.index)
      .map(|t| format!("spotify:track:{}", t.id))
  }
  fn next_index(&self, natural: bool) -> Option<usize> {
    if self.tracks.is_empty() {
      return None;
    }
    if natural && self.repeat == 2 {
      return Some(self.index);
    }
    if self.index + 1 < self.tracks.len() {
      Some(self.index + 1)
    } else if self.repeat == 1 {
      Some(0)
    } else {
      None
    }
  }
  fn append(&mut self, tracks: Vec<Track>, next: bool) -> Result<()> {
    if self.tracks.len() + tracks.len() > MAX_QUEUE {
      bail!("Queue limit is {MAX_QUEUE} tracks; remove some first");
    }
    let at = if next && !self.tracks.is_empty() {
      self.index + 1
    } else {
      self.tracks.len()
    };
    self.tracks.splice(at..at, tracks);
    Ok(())
  }
}

struct Backend {
  soloist: Mutex<Option<Arc<soloist::Player>>>,
  queue: Mutex<Queue>,
  player: Mutex<Option<Arc<StreamingPlayer>>>,
  // Commands that change queue ownership are serialized. Catalog calls and
  // state polling remain independent so a slow search cannot stall transport.
  commands: Mutex<()>,
  login: Mutex<Option<credentials::PendingLogin>>,
  artist_cursors: Mutex<std::collections::BTreeMap<usize, String>>,
}

fn state_path() -> Result<std::path::PathBuf> {
  Ok(
    crate::core::paths::app_state_dir()
      .ok_or_else(|| anyhow!("No state directory"))?
      .join("pomme_queue.json"),
  )
}

impl Backend {
  fn new() -> Self {
    let mut queue = state_path()
      .ok()
      .and_then(|p| {
        if std::fs::metadata(&p).ok()?.len() > 8 * 1024 * 1024 {
          return None;
        }
        serde_json::from_slice::<Queue>(&std::fs::read(p).ok()?).ok()
      })
      .unwrap_or_else(|| Queue {
        volume: 70,
        ..Queue::default()
      });
    queue.tracks.truncate(MAX_QUEUE);
    queue.index = queue.index.min(queue.tracks.len().saturating_sub(1));
    queue.volume = queue.volume.min(100);
    queue.repeat %= 3;
    Self {
      queue: Mutex::new(queue),
      soloist: Mutex::new(None),
      player: Mutex::new(None),
      commands: Mutex::new(()),
      login: Mutex::new(None),
      artist_cursors: Mutex::new(std::collections::BTreeMap::from([(0, String::new())])),
    }
  }

  async fn save(&self) -> Result<()> {
    let path = state_path()?;
    crate::core::paths::ensure_private_dir(path.parent().context("No state parent")?)?;
    let mut q = self.queue.lock().await;
    let position = q.position;
    q.position = q.position_ms();
    let bytes = serde_json::to_vec(&*q)?;
    q.position = position;
    drop(q);
    crate::core::auth::write_private_file_atomic(&path, &bytes)
  }

  async fn ensure_player(self: &Arc<Self>) -> Result<Arc<StreamingPlayer>> {
    if self.soloist.lock().await.is_some() {
      bail!("Soloist owns playback");
    }
    let mut slot = self.player.lock().await;
    if let Some(player) = slot.as_ref() {
      return Ok(player.clone());
    }
    let volume = self.queue.lock().await.volume;
    let player = Arc::new(
      tokio::time::timeout(
        Duration::from_secs(50),
        StreamingPlayer::new_cache_only(StreamingConfig {
          device_name: "Lime TUI".into(),
          bitrate: 160,
          audio_cache: false,
          autoplay: false,
          initial_volume: volume,
          ..StreamingConfig::default()
        }),
      )
      .await
      .context("Spotify audio initialization timed out")??,
    );
    let events = player.get_event_channel();
    *slot = Some(player.clone());
    let backend = self.clone();
    let active = player.clone();
    tokio::spawn(async move {
      backend.events(active, events).await;
    });
    Ok(player)
  }

  async fn events(
    self: Arc<Self>,
    player: Arc<StreamingPlayer>,
    mut events: librespot_playback::player::PlayerEventChannel,
  ) {
    while let Some(event) = events.recv().await {
      // A retired engine may still deliver buffered events.
      if !self
        .player
        .lock()
        .await
        .as_ref()
        .is_some_and(|p| Arc::ptr_eq(p, &player))
      {
        break;
      }
      let mut q = self.queue.lock().await;
      let (name, uri) = match &event {
        PlayerEvent::Playing { track_id, .. } => ("playing", Some(track_id.to_string())),
        PlayerEvent::Paused { track_id, .. } => ("paused", Some(track_id.to_string())),
        PlayerEvent::Loading { track_id, .. } => ("loading", Some(track_id.to_string())),
        PlayerEvent::Stopped { track_id, .. } => ("stopped", Some(track_id.to_string())),
        PlayerEvent::Unavailable { track_id, .. } => ("unavailable", Some(track_id.to_string())),
        PlayerEvent::EndOfTrack { track_id, .. } => ("end_of_track", Some(track_id.to_string())),
        PlayerEvent::TrackChanged { audio_item } => {
          ("track_changed", Some(audio_item.track_id.to_string()))
        }
        PlayerEvent::PositionChanged { track_id, .. } => ("progress", Some(track_id.to_string())),
        PlayerEvent::SessionDisconnected { reason, .. } => {
          use librespot_playback::player::SessionDisconnectReason::*;
          (
            match reason {
              ExternalDeviceHandoff => "handoff",
              UnexpectedShutdown => "session_lost",
              LocalCommand => "local_disconnect",
            },
            None,
          )
        }
        _ => ("other", None),
      };
      if name != "other" {
        let current = uri.map(|uri| q.current_uri().as_ref() == Some(&uri));
        let position = q.position_ms();
        q.trace
          .push(name, event.get_play_request_id(), current, position);
      }
      match event {
        PlayerEvent::PlayRequestIdChanged { play_request_id } => {
          q.observed_request = Some(play_request_id);
        }
        PlayerEvent::Loading {
          track_id,
          play_request_id,
          position_ms,
        } => {
          if q.observe_connect_track(&track_id.to_string(), play_request_id) {
            q.position = position_ms as u64;
            q.clock = None;
            q.playing = false;
            q.loading = true;
          }
        }
        PlayerEvent::Playing {
          track_id,
          position_ms,
          play_request_id,
        } => {
          if q.observe_connect_track(&track_id.to_string(), play_request_id)
            && !q.retry_pending
            && (q.play_intent || q.error.is_empty())
          {
            q.request_id = Some(play_request_id);
            q.reset_clock(position_ms as u64, true);
            q.error.clear();
          }
        }
        PlayerEvent::Paused {
          track_id,
          position_ms,
          play_request_id,
        } => {
          if q.observe_connect_track(&track_id.to_string(), play_request_id) {
            q.retry_pending = false;
            q.play_intent = false;
            q.reset_clock(position_ms as u64, false);
          }
        }
        PlayerEvent::Seeked {
          track_id,
          position_ms,
          play_request_id,
        }
        | PlayerEvent::PositionChanged {
          track_id,
          position_ms,
          play_request_id,
        } => {
          if q.owns_event(&track_id.to_string(), play_request_id) && !q.retry_pending {
            let playing = q.playing;
            q.reset_clock(position_ms as u64, playing);
          }
        }
        PlayerEvent::TrackChanged { audio_item } => {
          let uri = audio_item.track_id.to_string();
          if q
            .observed_request
            .is_some_and(|request| q.observe_connect_track(&uri, request))
          {
            let index = q.index;
            if let Some(track) = q.tracks.get_mut(index) {
              use librespot_metadata::audio::item::UniqueFields;
              if let UniqueFields::Track { artists, album, .. } = &audio_item.unique_fields {
                track.artist = artists
                  .0
                  .iter()
                  .map(|a| a.name.as_str())
                  .collect::<Vec<_>>()
                  .join(", ");
                track.album = album.clone();
              }
              track.title = audio_item.name.clone();
              track.art = audio_item
                .covers
                .first()
                .map(|c| c.url.clone())
                .unwrap_or_default();
              track.dur_ms = audio_item.duration_ms as u64;
            }
          }
        }
        PlayerEvent::EndOfTrack {
          track_id,
          play_request_id,
        } => {
          if !q.owns_event(&track_id.to_string(), play_request_id)
            || q.retry_pending
            || (!q.play_intent && !q.playing && !q.loading)
          {
            continue;
          }
          let position = q.position_ms();
          let duration = q.tracks.get(q.index).map(|t| t.dur_ms).unwrap_or(0);
          match classify_track_end(position, duration, q.play_intent, q.retry_used) {
            TrackEnd::RetryStart => {
              q.freeze();
              q.loading = true;
              q.retry_pending = true;
              q.retry_used = true;
              q.play_intent = true;
              q.trace
                .push("early_end", Some(play_request_id), Some(true), position);
              let generation = q.generation;
              let index = q.index;
              let backend = self.clone();
              let active = player.clone();
              // Stop the failed decoder immediately; otherwise packet errors can
              // repeatedly emit EndOfTrack while the recovery delay is pending.
              player.stop();
              tokio::spawn(async move {
                tokio::time::sleep(Duration::from_millis(500)).await;
                let _guard = backend.commands.lock().await;
                if !backend
                  .player
                  .lock()
                  .await
                  .as_ref()
                  .is_some_and(|p| Arc::ptr_eq(p, &active))
                {
                  return;
                }
                let q = backend.queue.lock().await;
                if !q.retry_pending
                  || !q.play_intent
                  || q.generation != generation
                  || q.index != index
                {
                  return;
                }
                drop(q);
                if let Err(error) = backend.load_index(&active, index, true).await {
                  backend.playback_error(error).await;
                }
              });
              continue;
            }
            TrackEnd::Failed => {
              q.freeze();
              q.loading = false;
              q.error="The Spotify audio stream ended before the song finished. Press Space to retry or R to reconnect.".into();
              q.trace
                .push("stream_failed", Some(play_request_id), Some(true), position);
              drop(q);
              player.stop();
              continue;
            }
            TrackEnd::Complete => {}
          }
          q.freeze();
          if q.connect_queue {
            // Spirc owns the phone's context and advances it itself.
            continue;
          }
          drop(q);
          let _guard = self.commands.lock().await;
          let q = self.queue.lock().await;
          if !q.owns_event(&track_id.to_string(), play_request_id) {
            continue;
          }
          if let Some(index) = q.next_index(true) {
            drop(q);
            if let Err(error) = self.load_index(&player, index, false).await {
              self.playback_error(error).await;
            }
          }
          continue;
        }
        PlayerEvent::Unavailable {
          track_id,
          play_request_id,
          failure,
        } => {
          q.trace.annotate_load_failure(failure);
          if q.owns_event(&track_id.to_string(), play_request_id) {
            q.freeze();
            q.loading = false;
            q.error = load_failure_message(failure.stage).into();
          }
        }
        PlayerEvent::Stopped {
          track_id,
          play_request_id,
        } if q.owns_event(&track_id.to_string(), play_request_id) && !q.retry_pending => {
          let unexpected = q.play_intent && (q.playing || q.loading);
          q.freeze();
          q.loading = false;
          if unexpected && q.error.is_empty() {
            q.error="Spotify stopped playback unexpectedly. Select the track again or press R to reconnect.".into();
          }
        }
        PlayerEvent::SessionDisconnected {
          reason: librespot_playback::player::SessionDisconnectReason::ExternalDeviceHandoff,
          ..
        } => {
          q.freeze();
          q.error="Spotify playback moved to another device. Press R to reconnect if you want to play here.".into();
        }
        _ => {}
      }
    }
  }

  async fn playback_error(&self, error: anyhow::Error) {
    let mut q = self.queue.lock().await;
    q.freeze();
    q.loading = false;
    q.error = error.to_string();
  }

  async fn start_index(self: &Arc<Self>, index: usize) -> Result<()> {
    {
      let mut q = self.queue.lock().await;
      if index >= q.tracks.len() {
        bail!("Queue position is out of range");
      }
      q.freeze();
      q.index = index;
      q.position = 0;
      q.loading = true;
      q.error.clear();
      q.trace.push("request_play", None, Some(true), 0);
    }
    let result = async {
      let official = self.soloist.lock().await.clone();
      if let Some(player) = official {
        soloist::load(self, &player, index).await
      } else {
        let player = self.ensure_player().await?;
        self.load_index(&player, index, false).await
      }
    }
    .await;
    if let Err(error) = result {
      let message = error.to_string();
      self.playback_error(error).await;
      bail!(message);
    }
    self.save().await?;
    Ok(())
  }

  async fn load_index(
    &self,
    player: &Arc<StreamingPlayer>,
    index: usize,
    retry: bool,
  ) -> Result<()> {
    let mut q = self.queue.lock().await;
    if index >= q.tracks.len() {
      bail!("Queue position is out of range");
    }
    q.freeze();
    q.index = index;
    q.position = 0;
    q.loading = true;
    q.play_intent = true;
    q.retry_used = retry;
    q.generation = q.generation.wrapping_add(1);
    q.request_id = None;
    q.observed_request = None;
    if !retry {
      q.connect_queue = false;
    }
    q.error.clear();
    let uri = q.current_uri().unwrap();
    player.activate();
    q.request_id = player.play_uri_tracked(&uri).await?;
    let request = q.request_id;
    q.trace.push(
      if retry { "retry_start" } else { "load_sent" },
      request,
      Some(true),
      0,
    );
    Ok(())
  }

  async fn state(&self) -> Value {
    let official = self.soloist.lock().await.clone();
    let player = self.player.lock().await.clone();
    let mut q = self.queue.lock().await;
    let recovering = player.as_ref().is_some_and(|p| p.is_recovering());
    if let Some(player) = player.as_ref() {
      if let Some(error) =
        player.take_audio_backend_error(crate::core::i18n::Language::En.strings())
      {
        q.freeze();
        q.loading = false;
        q.error = error;
      } else if !player.is_connected() && !recovering {
        q.freeze();
        q.loading = false;
        q.error = "Spotify connection ended. Press R to reconnect, then Space to resume.".into();
      }
    }
    json!({"authed": credentials::streaming_credentials_are_cached().unwrap_or(false),
            "playing": q.playing, "initializing": q.loading || recovering, "playbackEngine": if official.is_some() {"soloist"} else {"native"}, "posMs": q.position_ms(),
            "durMs": q.tracks.get(q.index).map(|t| t.dur_ms).unwrap_or(0),
            "volume": q.volume, "shuffle": q.shuffle, "repeat": q.repeat,
            "now": q.tracks.get(q.index), "queue": q.tracks, "queuePos": q.index, "err": q.error})
  }

  async fn handle(self: &Arc<Self>, method: &str, p: &Value) -> Result<Value> {
    match method {
      "hello" => {
        return Ok(
          json!({"protocol":1, "authed":credentials::streaming_credentials_are_cached()?}),
        );
      }
      "soloist_start" => {
        soloist::start(self, p).await?;
        return Ok(json!({}));
      }
      "state" => return Ok(self.state().await),
      "playback_diagnostics" => {
        let official = self.soloist.lock().await.clone();
        let player = self.player.lock().await.clone();
        let q = self.queue.lock().await;
        return Ok(
          json!({"playing":q.playing,"loading":q.loading,"positionMs":q.position_ms(),"hasError":!q.error.is_empty(),"playbackEngine":if official.is_some(){"soloist"}else{"native"},"connected":official.as_ref().map(|p|p.connected.load(std::sync::atomic::Ordering::Acquire)).unwrap_or_else(||player.as_ref().is_some_and(|p|p.is_connected())),"recovering":player.as_ref().is_some_and(|p|p.is_recovering()),"events":q.trace.entries}),
        );
      }
      "recent_history" => {
        let history = spotify::account_history::read().await;
        return Ok(
          json!({"recent":map_items(&history,"songs"),"err":history["err"],"retryAfterSeconds":history["retryAfterSeconds"]}),
        );
      }
      "spectrum" => {
        if self.soloist.lock().await.is_some() {
          bail!("Soloist audio analysis is not available");
        }
        let q = self.queue.lock().await;
        let active = q.playing && !q.loading;
        drop(q);
        return Ok(serde_json::to_value(crate::live_spectrum::bands(active))?);
      }
      "login_begin" => {
        let pending = credentials::begin_login().await?;
        let result = json!({"url": pending.url(), "code": pending.code()});
        let _ = open::that_detached(pending.url());
        *self.login.lock().await = Some(pending);
        return Ok(result);
      }
      "login_wait" => {
        let pending = self
          .login
          .lock()
          .await
          .take()
          .context("Start Spotify pairing first")?;
        let login = pending.finish().await?;
        spotify::account_history::clear();
        session::log_in(login)
          .await
          .map_err(|reason| anyhow!(reason.message()))?;
        return Ok(json!({}));
      }
      "account" => {
        let me = get("me", &[]).await?;
        return Ok(
          json!({"name":me["display_name"], "handle":me["id"], "country":me["country"], "storefront":"Spotify Premium"}),
        );
      }
      "lyrics" => {
        let id = text(p, "id");
        validate_id(id)?;
        let value = tokio::time::timeout(
          Duration::from_secs(8),
          spotify::route(&Method::GET, &format!("tracks/{id}/lyrics"), &[], None),
        )
        .await
        .context("Spotify lyrics request timed out")??;
        let Some(lyrics) = spotify::lyrics::parse_lyrics(value) else {
          return Ok(json!({"synced":false,"lines":[]}));
        };
        let lines: Vec<Value> = lyrics
          .lines
          .into_iter()
          .map(|(at, text)| json!({"atMs":at,"text":text}))
          .collect();
        return Ok(json!({"synced":lyrics.synced,"lines":lines}));
      }
      "artwork" => {
        let id = text(p, "id");
        validate_id(id)?;
        let kind = match text(p, "kind") {
          "song" | "songs" | "track" | "" => "tracks",
          "album" | "albums" => "albums",
          "playlist" | "playlists" => "playlists",
          "artist" | "artists" => "artists",
          _ => bail!("Unsupported artwork type"),
        };
        let value = get(&format!("{kind}/{id}"), &[]).await?;
        let mut url = art_url(&value);
        if url.is_empty() && kind == "playlists" {
          let page = get(
            &format!("playlists/{id}/items"),
            &[("limit", "4".into()), ("offset", "0".into())],
          )
          .await?;
          for entry in page["items"].as_array().into_iter().flatten() {
            url = art_url(&entry["item"]);
            if !url.is_empty() {
              break;
            }
          }
        }
        return Ok(json!({"url":url}));
      }
      "search" => {
        let term = text(p, "query");
        if term.trim().is_empty() {
          bail!("Enter a search term");
        }
        let value = get(
          "search",
          &[
            ("q", term.into()),
            ("type", "track,album,playlist,artist".into()),
            ("limit", "50".into()),
          ],
        )
        .await?;
        return Ok(
          json!({"songs": map_items(&value["tracks"],"songs"), "albums":map_items(&value["albums"],"albums"),
                    "playlists":map_items(&value["playlists"],"playlists"), "artists":map_items(&value["artists"],"artists")}),
        );
      }
      "library" => return self.library("songs", 0).await,
      "library_page" => {
        return self
          .library(text(p, "category"), number(p, "offset") as usize)
          .await;
      }
      "artist_page" => {
        let tracks = self.resolve("artists", text(p, "id")).await?;
        return Ok(json!({"songs":tracks}));
      }
      "discovery_item" => {
        return Ok(serde_json::to_value(
          self.resolve(text(p, "kind"), text(p, "id")).await?,
        )?);
      }
      "discovery" => {
        let page = get("views/made-for-x", &[]).await?;
        let sections: Vec<Value> = page["sections"]
          .as_array()
          .into_iter()
          .flatten()
          .take(20)
          .map(|s| {
            let items: Vec<Track> = s["playlists"]
              .as_array()
              .into_iter()
              .flatten()
              .take(50)
              .filter_map(|v| {
                let id = v["uri"].as_str()?.strip_prefix("spotify:playlist:")?;
                Some(Track {
                  id: id.into(),
                  title: text(v, "name").into(),
                  kind: "playlist".into(),
                  art: text(v, "image_url").into(),
                  ..Track::default()
                })
              })
              .collect();
            json!({"title":s["title"],"items":items})
          })
          .collect();
        return Ok(json!({"sections":sections}));
      }
      "save_item" => {
        let uri = spotify_uri(text(p, "kind"), text(p, "id"))?;
        spotify::route(&Method::PUT, "me/library", &[("uris", uri)], None).await?;
        return Ok(json!({}));
      }
      "save_playlist" => {
        let uris: Vec<String> = self
          .queue
          .lock()
          .await
          .tracks
          .iter()
          .map(|t| format!("spotify:track:{}", t.id))
          .collect();
        if uris.is_empty() {
          bail!("Queue is empty");
        }
        let playlist = spotify::route(
          &Method::POST,
          "me/playlists",
          &[],
          Some(&json!({"name":text(p,"name"),"public":false})),
        )
        .await?;
        let id = text(&playlist, "id");
        if id.is_empty() {
          bail!("Spotify returned no playlist ID");
        }
        for chunk in uris.chunks(100) {
          spotify::route(
            &Method::POST,
            &format!("playlists/{id}/items"),
            &[],
            Some(&json!({"uris":chunk})),
          )
          .await
          .context("Playlist was created but not all tracks could be added")?;
        }
        return Ok(json!({}));
      }
      _ => {}
    }
    // Fetch catalog items before taking the transport lock.
    let resolved = if matches!(method, "play" | "queue_next" | "queue_later") {
      Some(self.resolve(text(p, "kind"), text(p, "id")).await?)
    } else {
      None
    };
    let _guard = self.commands.lock().await;
    let official = self.soloist.lock().await.clone();
    match method {
      "play" => {
        let tracks = resolved.unwrap();
        if tracks.is_empty() {
          bail!("This item has no playable Spotify tracks");
        }
        {
          let mut q = self.queue.lock().await;
          q.freeze();
          q.tracks = tracks;
          q.index = 0;
          q.position = 0;
        }
        self.start_index(0).await?;
      }
      "queue_next" | "queue_later" => {
        self
          .queue
          .lock()
          .await
          .append(resolved.unwrap(), method == "queue_next")?;
      }
      "jump" => {
        self.start_index(number(p, "index") as usize).await?;
      }
      "next" => {
        let index = self.queue.lock().await.next_index(false);
        if let Some(index) = index {
          self.start_index(index).await?;
        } else {
          if let Some(player) = official.as_ref() {
            player.send(soloist::command("pause"))?;
          } else if let Some(player) = self.player.lock().await.as_ref() {
            player.pause();
          }
          self.queue.lock().await.freeze();
        }
      }
      "prev" => {
        let q = self.queue.lock().await;
        let index = if q.position_ms() > 3000 {
          q.index
        } else {
          q.index.saturating_sub(1)
        };
        drop(q);
        self.start_index(index).await?;
      }
      "play_pause" | "pause" => {
        let mut q = self.queue.lock().await;
        if q.tracks.is_empty() {
          bail!("Search for a track and press Enter to play");
        }
        if q.playing || method == "pause" {
          let position = q.position_ms();
          q.trace.push("request_pause", None, Some(true), position);
          q.freeze();
          drop(q);
          if let Some(player) = official.as_ref() {
            player.send(soloist::command("pause"))?;
          } else if let Some(player) = self.player.lock().await.as_ref() {
            player.pause();
          }
        } else {
          let index = q.index;
          let position = q.position;
          let restart = !q.error.is_empty() || (official.is_some() && !q.soloist_track_confirmed);
          q.play_intent = true;
          drop(q);
          let existing = self.player.lock().await.as_ref().cloned();
          if restart {
            self.start_index(index).await?;
          } else if let Some(player) = official.as_ref() {
            player.send(soloist::command("activate"))?;
            player.send(soloist::command("play"))?;
          } else if let Some(player) = existing {
            player.play();
          } else {
            self.start_index(index).await?;
            if let Some(player) = self.player.lock().await.as_ref() {
              player.seek(position.min(u32::MAX as u64) as u32);
            }
          }
        }
      }
      "seek" => {
        let mut q = self.queue.lock().await;
        q.retry_pending = false;
        let at = number(p, "atMs").min(q.tracks.get(q.index).map(|t| t.dur_ms).unwrap_or(0));
        let playing = q.playing;
        q.reset_clock(at, playing);
        drop(q);
        if let Some(player) = official.as_ref() {
          player.send(json!({"type":"command","command":"seek","position_ms":at}))?;
        } else if let Some(player) = self.player.lock().await.as_ref() {
          player.seek(at.min(u32::MAX as u64) as u32);
        }
      }
      "set_volume" => {
        let volume = number(p, "volume").min(100) as u8;
        self.queue.lock().await.volume = volume;
        if let Some(player) = official.as_ref() {
          player.send(json!({"type":"command","command":"set_volume","volume":volume}))?;
        } else if let Some(player) = self.player.lock().await.as_ref() {
          player.set_volume(volume);
        }
      }
      "toggle_shuffle" => {
        use rand::seq::SliceRandom;
        let mut q = self.queue.lock().await;
        q.shuffle = !q.shuffle;
        if q.shuffle {
          let start = (q.index + 1).min(q.tracks.len());
          q.tracks[start..].shuffle(&mut rand::rng());
        }
      }
      "cycle_repeat" => {
        let mut q = self.queue.lock().await;
        q.repeat = (q.repeat + 1) % 3;
      }
      "queue_edit" => {
        let mut q = self.queue.lock().await;
        let index = number(p, "index") as usize;
        let target = number(p, "target") as usize;
        match text(p, "action") {
          "clear" => {
            q.freeze();
            q.tracks.clear();
            q.index = 0;
            q.position = 0;
            drop(q);
            if let Some(player) = official.as_ref() {
              player.send(soloist::command("pause"))?;
            } else if let Some(player) = self.player.lock().await.as_ref() {
              player.stop();
            }
          }
          "remove" => {
            if index >= q.tracks.len() {
              bail!("Queue position is out of range");
            }
            let current = index == q.index;
            let playing = q.playing;
            q.tracks.remove(index);
            if index < q.index {
              q.index -= 1;
            }
            q.index = q.index.min(q.tracks.len().saturating_sub(1));
            if current {
              q.freeze();
              q.position = 0;
              let next = q.index;
              let empty = q.tracks.is_empty();
              drop(q);
              if let Some(player) = official.as_ref() {
                player.send(soloist::command("pause"))?;
              } else if let Some(player) = self.player.lock().await.as_ref() {
                player.stop();
              }
              if !empty && playing {
                self.start_index(next).await?;
              }
            }
          }
          "move" => {
            if index >= q.tracks.len() || target >= q.tracks.len() {
              bail!("Queue position is out of range");
            }
            let track = q.tracks.remove(index);
            q.tracks.insert(target, track);
            q.index = if q.index == index {
              target
            } else if index < q.index && target >= q.index {
              q.index - 1
            } else if index > q.index && target <= q.index {
              q.index + 1
            } else {
              q.index
            };
          }
          _ => bail!("Unknown queue edit"),
        }
      }
      "reload" => {
        if official.is_some() {
          bail!("Restart Soloist through Lime's reconnect control");
        }
        if let Some(player) = self.player.lock().await.take() {
          player.shutdown();
        }
        {
          let mut q = self.queue.lock().await;
          q.freeze();
          q.request_id = None;
          q.generation = q.generation.wrapping_add(1);
        }
        self.ensure_player().await?;
        let q = self.queue.lock().await;
        let uri = q.current_uri();
        let position = q.position;
        drop(q);
        if let Some(uri) = uri {
          let player = self.ensure_player().await?;
          player.load_uri_paused(&uri).await?;
          player.seek(position.min(u32::MAX as u64) as u32);
        }
      }
      "sign_out" => {
        if let Some(player) = self.soloist.lock().await.take() {
          player.shutdown().await;
        }
        soloist::clear_session()?;
        spotify::account_history::clear();
        if let Some(player) = self.player.lock().await.take() {
          player.shutdown();
        }
        session::forget().await;
        if let Some(path) = credentials::get_default_cache_path() {
          for name in ["credentials.json", "credentials.dpapi"] {
            if path.join(name).try_exists()? {
              bail!("Could not remove cached Spotify login");
            }
          }
        }
        *self.queue.lock().await = Queue {
          volume: 70,
          ..Queue::default()
        };
      }
      _ => bail!("Unknown Spotify engine method: {method}"),
    }
    self.save().await?;
    Ok(json!({}))
  }

  async fn library(&self, category: &str, offset: usize) -> Result<Value> {
    if category == "artists" {
      let target = offset.min(10000);
      let mut cursors = self.artist_cursors.lock().await;
      if target == 0 {
        cursors.clear();
        cursors.insert(0, String::new());
      }
      let (mut page_offset, mut after) = cursors
        .range(..=target)
        .next_back()
        .map(|(n, s)| (*n, s.clone()))
        .unwrap_or((0, String::new()));
      loop {
        let mut query = vec![("type", "artist".into()), ("limit", "50".into())];
        if !after.is_empty() {
          query.push(("after", after));
        }
        let value = get("me/following", &query).await?;
        let artists = map_items(&value["artists"], "artists");
        if let Some(cursor) = value["artists"]["cursors"]["after"].as_str() {
          after = cursor.into();
          if artists.len() == 50 {
            cursors.insert(page_offset + 50, after.clone());
          }
        } else {
          after = String::new();
        }
        if page_offset >= target {
          return Ok(json!({"artists": artists}));
        }
        if artists.len() < 50 || after.is_empty() {
          return Ok(json!({"artists": []}));
        }
        page_offset += 50;
      }
    }
    let paging = [
      ("limit", "50".into()),
      ("offset", offset.min(10000).to_string()),
    ];
    let (path, kind) = match category {
      "songs" | "favorites" => ("me/tracks", "songs"),
      "albums" => ("me/albums", "albums"),
      "playlists" => ("me/playlists", "playlists"),
      "artists" => ("me/following", "artists"),
      _ => bail!("Unknown library category"),
    };
    let mut query = paging.to_vec();
    if category == "artists" {
      query.push(("type", "artist".into()));
    }
    let value = get(path, &query).await?;
    let page = if category == "artists" {
      &value["artists"]
    } else {
      &value
    };
    Ok(json!({kind:map_items(page,kind)}))
  }

  async fn resolve(&self, kind: &str, id: &str) -> Result<Vec<Track>> {
    validate_id(id)?;
    if matches!(kind, "songs" | "song" | "track" | "") {
      let value = get(&format!("tracks/{id}"), &[]).await?;
      return Ok(vec![
        map_track(&value, "songs").context("Spotify returned no track")?
      ]);
    }
    let (path, key) = match kind {
      "albums" | "album" => (format!("albums/{id}/tracks"), "items"),
      "playlists" | "playlist" => (format!("playlists/{id}/items"), "items"),
      "artists" | "artist" => (format!("artists/{id}/top-tracks"), "tracks"),
      _ => bail!("Unsupported Spotify item type"),
    };
    let mut tracks = Vec::new();
    let mut offset = 0;
    loop {
      let page = get(
        &path,
        &[("offset", offset.to_string()), ("limit", "50".into())],
      )
      .await?;
      let items = page[key].as_array().cloned().unwrap_or_default();
      let count = items.len();
      for item in items {
        if let Some(track) = map_track(&item, "songs") {
          tracks.push(track);
        }
      }
      if tracks.len() >= MAX_QUEUE {
        tracks.truncate(MAX_QUEUE);
        break;
      }
      offset += count;
      if key == "tracks"
        || count < 50
        || offset >= MAX_QUEUE
        || page["total"]
          .as_u64()
          .is_some_and(|total| offset as u64 >= total)
      {
        break;
      }
    }
    if matches!(kind, "albums" | "album") {
      let album = get(&format!("albums/{id}"), &[]).await?;
      let art = art_url(&album);
      for track in &mut tracks {
        track.album = text(&album, "name").into();
        if track.art.is_empty() {
          track.art = art.clone();
        }
      }
    }
    Ok(tracks)
  }
}

fn text<'a>(v: &'a Value, key: &str) -> &'a str {
  v[key].as_str().unwrap_or("")
}
fn number(v: &Value, key: &str) -> u64 {
  v[key].as_u64().unwrap_or(0)
}
fn validate_id(id: &str) -> Result<()> {
  if id.len() != 22 || !id.bytes().all(|b| b.is_ascii_alphanumeric()) {
    bail!("Invalid Spotify ID");
  }
  Ok(())
}
fn spotify_uri(kind: &str, id: &str) -> Result<String> {
  validate_id(id)?;
  let kind = match kind {
    "songs" | "song" | "track" => "track",
    "albums" | "album" => "album",
    "playlists" | "playlist" => "playlist",
    "artists" | "artist" => "artist",
    _ => bail!("Unsupported Spotify item type"),
  };
  Ok(format!("spotify:{kind}:{id}"))
}
async fn get(path: &str, query: &[(&str, String)]) -> Result<Value> {
  tokio::time::timeout(
    Duration::from_secs(25),
    spotify::route(&Method::GET, path, query, None),
  )
  .await
  .context("Spotify request timed out")?
}
fn art_url(v: &Value) -> String {
  let images = v["images"]
    .as_array()
    .or_else(|| v["album"]["images"].as_array());
  images
    .and_then(|items| {
      items
        .iter()
        .filter(|i| i["url"].as_str().is_some())
        .min_by_key(|i| {
          let w = i["width"].as_u64().unwrap_or(300);
          w.abs_diff(256)
        })
    })
    .map(|i| text(i, "url").to_string())
    .unwrap_or_default()
}
fn map_track(v: &Value, kind: &str) -> Option<Track> {
  let v = if v["item"].is_object() {
    &v["item"]
  } else if v["track"].is_object() {
    &v["track"]
  } else if v["album"].is_object() && kind == "albums" {
    &v["album"]
  } else {
    v
  };
  let id = v["id"].as_str()?;
  if validate_id(id).is_err() || v["is_local"].as_bool() == Some(true) {
    return None;
  }
  let artist = v["artists"]
    .as_array()
    .map(|a| {
      a.iter()
        .filter_map(|v| v["name"].as_str())
        .collect::<Vec<_>>()
        .join(", ")
    })
    .unwrap_or_else(|| text(&v["owner"], "display_name").into());
  Some(Track {
    id: id.into(),
    title: text(v, "name").into(),
    artist,
    album: text(&v["album"], "name").into(),
    kind: match kind {
      "songs" | "track" => "song",
      "albums" => "album",
      "playlists" => "playlist",
      "artists" => "artist",
      other => other,
    }
    .into(),
    dur_ms: number(v, "duration_ms"),
    art: art_url(v),
  })
}
fn map_items(page: &Value, kind: &str) -> Vec<Track> {
  page["items"]
    .as_array()
    .into_iter()
    .flatten()
    .take(50)
    .filter_map(|v| map_track(v, kind))
    .collect()
}

#[derive(Deserialize)]
struct Request {
  id: u64,
  method: String,
  #[serde(default)]
  params: Value,
}

#[cfg(all(target_os = "windows", feature = "windows-media"))]
fn start_media(backend: Arc<Backend>) -> Option<Arc<smtc_tokio::WindowsMediaManager>> {
  let manager = Arc::new(smtc_tokio::WindowsMediaManager::new().ok()?);
  let mut events = manager.take_event_rx()?;
  let media = manager.clone();
  tokio::spawn(async move {
    use smtc_tokio::WindowsMediaEvent as Event;
    let mut tick = tokio::time::interval(Duration::from_secs(1));
    let mut last_id = String::new();
    loop {
      tokio::select! {
        _ = tick.tick() => {
          let q = backend.queue.lock().await;
          if let Some(track) = q.tracks.get(q.index) {
            if track.id != last_id {
              media.set_metadata(&track.title, std::slice::from_ref(&track.artist), &track.album, track.dur_ms, None);
              last_id = track.id.clone();
            }
            media.set_position(q.position_ms());
          } else if !last_id.is_empty() {
            media.set_metadata("Lime TUI", &[], "", 0, None);
            last_id.clear();
          }
          media.set_playback_status(q.playing);
        }
        event = events.recv() => {
          let Some(event) = event else {break;};
          let (method, params) = match event {
            Event::Play => {
              if backend.queue.lock().await.playing {continue;}
              ("play_pause", json!({}))
            }
            Event::Pause | Event::Stop => ("pause", json!({})),
            Event::Next => ("next", json!({})),
            Event::Previous => ("prev", json!({})),
            Event::SetPosition(position) => ("seek", json!({"atMs": position})),
          };
          if let Err(error) = backend.handle(method, &params).await {
            backend.queue.lock().await.error = error.to_string();
          }
        }
      }
    }
  });
  Some(manager)
}

pub async fn run() -> Result<()> {
  // No file logs or credential-bearing protocol diagnostics.
  crate::live_spectrum::install();
  let backend = Arc::new(Backend::new());
  #[cfg(all(target_os = "windows", feature = "windows-media"))]
  let _media = start_media(backend.clone());
  let (out_tx, mut out_rx) = mpsc::channel::<Value>(32);
  let writer = tokio::spawn(async move {
    while let Some(value) = out_rx.recv().await {
      let mut stdout = std::io::stdout().lock();
      if serde_json::to_writer(&mut stdout, &value).is_err()
        || writeln!(stdout).is_err()
        || stdout.flush().is_err()
      {
        break;
      }
    }
  });
  let permits = Arc::new(Semaphore::new(8));
  let (input_tx, mut input_rx) = mpsc::channel::<Result<Vec<u8>>>(16);
  std::thread::spawn(move || {
    use std::io::BufRead;
    let stdin = std::io::stdin();
    let mut reader = stdin.lock();
    let mut line = Vec::new();
    loop {
      // fill_buf prevents allocation beyond the protocol's line limit.
      let chunk = match reader.fill_buf() {
        Ok(c) => c,
        Err(e) => {
          let _ = input_tx.blocking_send(Err(e.into()));
          break;
        }
      };
      if chunk.is_empty() {
        if !line.is_empty() {
          let _ = input_tx.blocking_send(Err(anyhow!("Incomplete engine request")));
        }
        break;
      }
      let end = chunk.iter().position(|b| *b == b'\n').map(|i| i + 1);
      let size = end.unwrap_or(chunk.len());
      if line.len() + size > MAX_LINE {
        let _ = input_tx.blocking_send(Err(anyhow!("Request exceeds 64 KiB")));
        break;
      }
      line.extend_from_slice(&chunk[..size]);
      reader.consume(size);
      if end.is_some()
        && input_tx
          .blocking_send(Ok(std::mem::take(&mut line)))
          .is_err()
      {
        break;
      }
    }
  });
  let mut requests = tokio::task::JoinSet::new();
  let mut checkpoint = tokio::time::interval(Duration::from_secs(15));
  loop {
    let line = tokio::select! {
        result=input_rx.recv()=>match result {Some(Ok(line))=>line,Some(Err(error))=>return Err(error),None=>break},
        Some(_)=requests.join_next()=>continue,
        _=checkpoint.tick()=>{let _=backend.save().await;continue;}
    };
    let request: Request = match serde_json::from_slice(&line) {
      Ok(r) => r,
      Err(_) => {
        out_tx
          .send(json!({"id":0,"error":"Invalid engine request"}))
          .await?;
        continue;
      }
    };
    if request.method == "shutdown" {
      out_tx.send(json!({"id":request.id,"result":{}})).await?;
      break;
    }
    let permit = match permits.clone().try_acquire_owned() {
      Ok(permit) => permit,
      Err(_) => {
        out_tx
          .send(json!({"id":request.id,"error":"Spotify engine is busy; retry shortly"}))
          .await?;
        continue;
      }
    };
    let backend = backend.clone();
    let tx = out_tx.clone();
    requests.spawn(async move{
            let _permit=permit;
            let duration=if matches!(request.method.as_str(), "login_wait" | "save_playlist"){Duration::from_secs(300)}else{Duration::from_secs(55)};
            let result=tokio::time::timeout(duration,backend.handle(&request.method,&request.params)).await;
            let reply=match result{
                Ok(Ok(value))=>json!({"id":request.id,"result":value}),
                Ok(Err(error))=>json!({"id":request.id,"error":error.to_string()}),
                Err(_)=>json!({"id":request.id,"error":"Spotify operation timed out; retry or press R to reconnect"}),
            };
            let _=tx.send(reply).await;
        });
  }
  if let Some(player) = backend.soloist.lock().await.take() {
    player.shutdown().await;
  }
  requests.abort_all();
  while requests.join_next().await.is_some() {}
  if let Some(player) = backend.player.lock().await.take() {
    player.shutdown();
  }
  let _ = backend.save().await;
  drop(out_tx);
  // Runtime teardown cancels any in-flight network/login tasks on owner EOF.
  tokio::time::timeout(Duration::from_secs(1), writer)
    .await
    .ok();
  Ok(())
}

#[cfg(test)]
mod tests {
  use super::*;
  #[test]
  fn phone_track_changes_update_the_queue_and_reject_old_decoder_events() {
    let mut q = Queue {
      tracks: vec![track(1), track(2)],
      request_id: Some(7),
      retry_pending: true,
      retry_used: true,
      error: "old failure".into(),
      generation: 3,
      ..Queue::default()
    };
    let first = q.current_uri().unwrap();
    let second = format!("spotify:track:{}", q.tracks[1].id);
    assert!(!q.observe_connect_track(&second, 6));
    assert!(!q.observe_connect_track(&second, 7));
    assert!(q.observe_connect_track(&second, 8));
    assert_eq!(q.index, 1);
    assert_eq!(q.tracks.len(), 2);
    assert_eq!(q.generation, 4);
    assert!(q.connect_queue && q.play_intent);
    assert!(!q.retry_pending && !q.retry_used);
    assert!(q.error.is_empty());
    assert!(!q.owns_event(&first, 7));
    assert!(!q.observe_connect_track(&first, 7));
    assert!(q.owns_event(&second, 8));
    // Repeating the same track from the phone is a new decoder request too.
    assert!(q.observe_connect_track(&second, 9));
    assert!(!q.owns_event(&second, 8));
  }

  #[test]
  fn phone_selection_outside_the_queue_is_bounded_and_local_loads_stay_local() {
    let mut q = Queue {
      tracks: (0..MAX_QUEUE).map(track).collect(),
      request_id: Some(10),
      ..Queue::default()
    };
    let current = q.current_uri().unwrap();
    assert!(q.observe_connect_track(&current, 10));
    assert!(!q.connect_queue);
    let remote = format!("spotify:track:{:022}", MAX_QUEUE + 1);
    assert!(q.observe_connect_track(&remote, 11));
    assert_eq!(q.tracks.len(), MAX_QUEUE);
    assert_eq!(q.current_uri().as_deref(), Some(remote.as_str()));
    assert!(!q.observe_connect_track("spotify:episode:invalid", 12));
    assert_eq!(q.request_id, Some(11));
  }
  #[test]
  fn track_load_failures_preserve_only_typed_categories_and_do_not_persist() {
    use librespot_playback::player::{LoadFailure, LoadStage};
    let mut q = Queue::default();
    for stage in [
      LoadStage::Metadata,
      LoadStage::Availability,
      LoadStage::Format,
      LoadStage::Download,
      LoadStage::AudioKey,
      LoadStage::Decode,
      LoadStage::Seek,
      LoadStage::LocalFile,
      LoadStage::Internal,
    ] {
      q.trace.push("unavailable", Some(4), Some(true), 0);
      q.trace.annotate_load_failure(LoadFailure {
        stage,
        kind: Some(librespot_core::error::ErrorKind::PermissionDenied),
      });
      let event = q.trace.entries.back().unwrap();
      assert_eq!(event["failureStage"], stage.code());
      assert_eq!(event["failureKind"], 7);
      assert!(!load_failure_message(stage).is_empty());
    }
    let exported = serde_json::to_string(&q.trace.entries).unwrap();
    assert!(exported.contains("audio_key"));
    let stored = serde_json::to_string(&q).unwrap();
    assert!(!stored.contains("failureStage"));
    assert!(!stored.contains("PermissionDenied"));
  }

  #[test]
  fn playback_trace_is_bounded_and_never_persisted_with_the_queue() {
    let mut q = Queue::default();
    for i in 0..200 {
      q.trace.push("progress", Some(i), Some(true), i);
    }
    assert_eq!(q.trace.entries.len(), 64);
    let saved = serde_json::to_value(&q).unwrap();
    assert!(saved.get("trace").is_none());
    assert!(saved.to_string().find("progress").is_none());
  }
  #[test]
  fn premature_start_end_retries_once_and_does_not_advance_the_queue() {
    assert_eq!(
      classify_track_end(1000, 180000, true, false),
      TrackEnd::RetryStart
    );
    assert_eq!(
      classify_track_end(1000, 180000, true, true),
      TrackEnd::Failed
    );
    assert_eq!(
      classify_track_end(30000, 180000, true, false),
      TrackEnd::Failed
    );
    assert_eq!(
      classify_track_end(1000, 180000, false, false),
      TrackEnd::Failed
    );
    assert_eq!(
      classify_track_end(179000, 180000, true, false),
      TrackEnd::Complete
    );
    assert_eq!(classify_track_end(0, 0, true, false), TrackEnd::Complete);
  }
  #[test]
  fn old_decoder_events_for_the_same_song_do_not_own_a_new_play_request() {
    let q = Queue {
      tracks: vec![track(1)],
      request_id: Some(9),
      ..Queue::default()
    };
    let uri = q.current_uri().unwrap();
    assert!(!q.owns_event(&uri, 8));
    assert!(q.owns_event(&uri, 9));
  }
  #[test]
  fn pausing_cancels_a_scheduled_stream_retry() {
    let mut q = Queue {
      retry_pending: true,
      play_intent: true,
      loading: true,
      ..Queue::default()
    };
    q.freeze();
    assert!(!q.retry_pending && !q.play_intent && !q.loading);
  }
  #[tokio::test]
  #[ignore = "requires cached Premium login, isolated state and LIME_TEST_TRACK_IDS_FILE"]
  async fn native_track_start_batch_stays_playing() {
    session::ALLOW_IN_TESTS.store(true, std::sync::atomic::Ordering::Relaxed);
    let ids: Vec<String> = serde_json::from_slice(
      &std::fs::read(std::env::var("LIME_TEST_TRACK_IDS_FILE").expect("track fixture path"))
        .unwrap(),
    )
    .unwrap();
    assert!(!ids.is_empty());
    let backend = Arc::new(Backend::new());
    backend
      .handle("set_volume", &json!({"volume":15}))
      .await
      .unwrap();
    let mut failures = 0;
    for (i, id) in ids.iter().enumerate() {
      if backend
        .handle("play", &json!({"kind":"song","id":id}))
        .await
        .is_err()
      {
        failures += 1;
        println!("sample {}: play request failed", i + 1);
        continue;
      }
      let deadline = Instant::now() + Duration::from_secs(35);
      let mut started = None;
      let mut failed = false;
      while Instant::now() < deadline {
        let state = backend.state().await;
        if state["playing"] == true {
          started.get_or_insert_with(Instant::now);
        }
        if !text(&state, "err").is_empty() || (started.is_some() && state["playing"] != true) {
          failed = true;
          break;
        }
        if started.is_some_and(|at| at.elapsed() > Duration::from_secs(7)) {
          break;
        }
        tokio::time::sleep(Duration::from_millis(100)).await;
      }
      if failed || started.is_none() {
        failures += 1;
        println!(
          "sample {}: stopped/failed; {}",
          i + 1,
          backend
            .handle("playback_diagnostics", &json!({}))
            .await
            .unwrap()
        );
      } else {
        println!("sample {}: stable for 7 seconds", i + 1);
      }
    }
    if let Some(player) = backend.player.lock().await.take() {
      player.shutdown();
    }
    assert_eq!(failures, 0, "native start failures in batch");
  }
  #[tokio::test]
  #[ignore = "requires cached Premium login, isolated state and LIME_TEST_TRACK_IDS_FILE"]
  async fn native_start_failure_retries_once_and_respects_pause() {
    session::ALLOW_IN_TESTS.store(true, std::sync::atomic::Ordering::Relaxed);
    let ids: Vec<String> = serde_json::from_slice(
      &std::fs::read(std::env::var("LIME_TEST_TRACK_IDS_FILE").unwrap()).unwrap(),
    )
    .unwrap();
    let backend = Arc::new(Backend::new());
    backend
      .handle("set_volume", &json!({"volume":15}))
      .await
      .unwrap();
    backend
      .handle("play", &json!({"kind":"song","id":ids[0]}))
      .await
      .unwrap();
    async fn playing(backend: &Arc<Backend>) {
      let deadline = Instant::now() + Duration::from_secs(35);
      while Instant::now() < deadline {
        if backend.queue.lock().await.playing {
          return;
        }
        tokio::time::sleep(Duration::from_millis(50)).await;
      }
      panic!("native playback did not start");
    }
    async fn end(backend: &Arc<Backend>, request: Option<u64>) {
      let player = backend.player.lock().await.as_ref().unwrap().clone();
      let q = backend.queue.lock().await;
      let uri = librespot_core::SpotifyUri::from_uri(&q.current_uri().unwrap()).unwrap();
      let id = request.or(q.request_id).unwrap();
      drop(q);
      let (tx, rx) = tokio::sync::mpsc::unbounded_channel();
      tx.send(PlayerEvent::EndOfTrack {
        track_id: uri,
        play_request_id: id,
      })
      .unwrap();
      drop(tx);
      backend.clone().events(player, rx).await;
    }
    playing(&backend).await;
    let first = backend.queue.lock().await.request_id.unwrap();
    end(&backend, None).await;
    tokio::time::sleep(Duration::from_millis(650)).await;
    playing(&backend).await;
    let second = backend.queue.lock().await.request_id.unwrap();
    assert_ne!(first, second);
    assert!(backend.queue.lock().await.retry_used);
    end(&backend, Some(first)).await;
    assert!(
      backend.queue.lock().await.playing,
      "stale end stopped the replacement request"
    );
    end(&backend, None).await;
    tokio::time::sleep(Duration::from_millis(650)).await;
    {
      let q = backend.queue.lock().await;
      assert!(!q.playing && !q.retry_pending && !q.error.is_empty());
      assert_eq!(q.request_id, Some(second));
      assert_eq!(q.index, 0);
    }
    {
      // A queued Playing notification from the failed decoder must not erase
      // the failure or make the UI appear to be playing after we stopped it.
      let player = backend.player.lock().await.as_ref().unwrap().clone();
      let uri =
        librespot_core::SpotifyUri::from_uri(&backend.queue.lock().await.current_uri().unwrap())
          .unwrap();
      let (tx, rx) = tokio::sync::mpsc::unbounded_channel();
      tx.send(PlayerEvent::Playing {
        track_id: uri,
        play_request_id: second,
        position_ms: 1000,
      })
      .unwrap();
      drop(tx);
      backend.clone().events(player, rx).await;
      let q = backend.queue.lock().await;
      assert!(
        !q.playing && !q.error.is_empty(),
        "late Playing hid the stopped decoder failure"
      );
    }
    backend.handle("play_pause", &json!({})).await.unwrap();
    playing(&backend).await;
    let third = backend.queue.lock().await.request_id;
    assert!(!backend.queue.lock().await.retry_used);
    end(&backend, None).await;
    backend.handle("pause", &json!({})).await.unwrap();
    tokio::time::sleep(Duration::from_millis(750)).await;
    {
      let q = backend.queue.lock().await;
      assert!(!q.playing && !q.retry_pending && !q.play_intent);
      assert_eq!(
        q.request_id, third,
        "pause was overwritten by an automatic retry"
      );
    }
    end(&backend, None).await;
    assert!(
      backend.queue.lock().await.error.is_empty(),
      "an end arriving after Pause created a false failure"
    );
    if let Some(player) = backend.player.lock().await.take() {
      player.shutdown();
    }
    println!(
      "Verified one retry, no queue skip, stale-end rejection, explicit resume and pause cancellation"
    );
  }
  #[test]
  fn playlist_item_envelope_maps_to_a_playable_song() {
    let value = json!({"item":{"id":"0123456789012345678901","name":"Fixture song","type":"track","duration_ms":120000,"artists":[{"name":"Fixture artist"}]}});
    let track = map_track(&value, "songs").expect("Spotify playlist item must not disappear");
    assert_eq!(track.kind, "song");
    assert_eq!(
      map_track(
        &json!({"id":"0123456789012345678901","name":"Album"}),
        "albums"
      )
      .unwrap()
      .kind,
      "album"
    );
  }
  fn track(n: usize) -> Track {
    Track {
      id: format!("{n:022}"),
      kind: "songs".into(),
      dur_ms: 10000,
      ..Track::default()
    }
  }
  #[test]
  fn queue_repeat_and_end_are_distinct() {
    let mut q = Queue {
      tracks: vec![track(1), track(2)],
      index: 1,
      ..Queue::default()
    };
    assert_eq!(q.next_index(true), None);
    q.shuffle = true;
    assert_eq!(q.next_index(true), None);
    q.repeat = 1;
    assert_eq!(q.next_index(true), Some(0));
    q.repeat = 2;
    assert_eq!(q.next_index(true), Some(1));
    assert_eq!(q.next_index(false), None);
  }
  #[test]
  fn queue_capacity_rejection_preserves_existing_tracks() {
    let mut q = Queue {
      tracks: (0..MAX_QUEUE).map(track).collect(),
      ..Queue::default()
    };
    assert!(q.append(vec![track(1001)], true).is_err());
    assert_eq!(q.tracks.len(), MAX_QUEUE);
  }
  #[test]
  fn next_insertion_keeps_album_order() {
    let mut q = Queue {
      tracks: vec![track(1), track(4)],
      ..Queue::default()
    };
    q.append(vec![track(2), track(3)], true).unwrap();
    assert_eq!(
      q.tracks.iter().map(|t| t.id.clone()).collect::<Vec<_>>(),
      (1..=4).map(|n| track(n).id).collect::<Vec<_>>()
    );
  }
  #[test]
  fn saved_track_mapping_selects_small_art_and_skips_local() {
    let value = json!({"track":{"id":"0123456789012345678901","name":"Song","duration_ms":9000,"artists":[{"name":"Artist"}],"album":{"name":"Album","images":[{"width":640,"url":"large"},{"width":300,"url":"small"}]}}});
    let t = map_track(&value, "songs").unwrap();
    assert_eq!(t.art, "small");
    assert_eq!(t.album, "Album");
    assert_eq!(t.artist, "Artist");
    assert!(map_track(&json!({"id":"local","is_local":true}), "songs").is_none());
  }
  #[test]
  fn restart_never_restores_playing_flag() {
    let q = Queue {
      playing: true,
      loading: true,
      tracks: vec![track(1)],
      ..Queue::default()
    };
    let restored: Queue = serde_json::from_slice(&serde_json::to_vec(&q).unwrap()).unwrap();
    assert!(!restored.playing);
    assert!(!restored.loading);
  }
}
