use super::{Ring, analyze_ring};
use anyhow::{Result, bail};
use std::{
  sync::{
    Arc, Mutex,
    atomic::{AtomicBool, Ordering},
  },
  thread::{self, JoinHandle},
  time::Duration,
};
use wasapi::{AudioClient, Direction, SampleType, StreamMode, WaveFormat};

pub struct SoloistCapture {
  ring: Arc<Mutex<Ring>>,
  stopped: Arc<AtomicBool>,
  worker: JoinHandle<()>,
}

impl SoloistCapture {
  pub fn start(pid: u32) -> std::io::Result<Self> {
    let ring = Arc::new(Mutex::new(Ring::new()));
    let stopped = Arc::new(AtomicBool::new(false));
    let worker = {
      let ring = ring.clone();
      let stopped = stopped.clone();
      thread::Builder::new()
        .name("soloist-spectrum".into())
        .spawn(move || {
          if wasapi::initialize_mta().is_ok() {
            let _ = capture(pid, &ring, &stopped);
            wasapi::deinitialize();
          }
        })?
    };
    Ok(Self {
      ring,
      stopped,
      worker,
    })
  }

  pub fn bands(&self) -> Result<[f64; 32]> {
    if self.stopped.load(Ordering::Relaxed) || self.worker.is_finished() {
      bail!("Soloist process audio capture is unavailable");
    }
    Ok(analyze_ring(&self.ring))
  }

  pub fn stop(&self) {
    self.stopped.store(true, Ordering::Relaxed);
    self.worker.thread().unpark();
  }
}

impl Drop for SoloistCapture {
  fn drop(&mut self) {
    self.stop();
  }
}

fn capture(pid: u32, ring: &Mutex<Ring>, stopped: &AtomicBool) -> Result<()> {
  // The PID belongs to the Soloist child we launched. Windows scopes the stream
  // to that process tree across output devices.
  let mut client = AudioClient::new_application_loopback_client(pid, true)?;
  let format = WaveFormat::new(32, 32, SampleType::Float, super::RATE as usize, 2, None);
  let mode = StreamMode::PollingShared {
    autoconvert: true,
    buffer_duration_hns: 0,
  };
  client.initialize_client(&format, Direction::Capture, mode, None)?;
  let reader = client.get_audiocaptureclient()?;
  let mut bytes = vec![0u8; 44_100 * 8];
  let mut samples = Vec::with_capacity(super::N * 2);
  client.start_stream()?;
  while !stopped.load(Ordering::Relaxed) {
    // Bound each drain so even a stalled consumer cannot prevent shutdown.
    for _ in 0..16 {
      let (frames, _) = reader.read_from_device(&mut bytes)?;
      if frames == 0 {
        break;
      }
      let packet = &bytes[..frames as usize * 8];
      let tail = &packet[packet.len().saturating_sub(super::N * 8)..];
      samples.clear();
      samples.extend(
        tail
          .as_chunks::<4>()
          .0
          .iter()
          .copied()
          .map(f32::from_le_bytes),
      );
      if let Ok(mut ring) = ring.try_lock() {
        ring.push(&samples);
      }
    }
    thread::park_timeout(Duration::from_millis(10));
  }
  client.stop_stream()?;
  Ok(())
}
