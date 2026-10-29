//! A small stereo FFT over audio actually consumed by Lime's output mixer.
//! No microphone/system capture, audio files, or unbounded sample queue.
use realfft::{num_complex::Complex, RealFftPlanner, RealToComplex};
use std::sync::{Arc, Mutex, OnceLock};
use std::time::{Duration, Instant};

const N: usize = 2048;
const RATE: f64 = 44100.0;
const STALE: Duration = Duration::from_millis(200);

struct Ring {
  samples: [[f32; N]; 2],
  cursor: usize,
  count: usize,
  updated: Option<Instant>,
}
impl Ring {
  fn new() -> Self {
    Self {
      samples: [[0.0; N]; 2],
      cursor: 0,
      count: 0,
      updated: None,
    }
  }
  fn push(&mut self, samples: &[f32]) {
    for frame in samples.as_chunks::<2>().0 {
      for (channel, value) in frame.iter().enumerate() {
        self.samples[channel][self.cursor] = if value.is_finite() {
          value.clamp(-1.0, 1.0)
        } else {
          0.0
        };
      }
      self.cursor = (self.cursor + 1) % N;
      self.count = (self.count + 1).min(N);
    }
    self.updated = Some(Instant::now());
  }
}
struct Analyzer {
  fft: Arc<dyn RealToComplex<f32>>,
  input: Vec<f32>,
  output: Vec<Complex<f32>>,
  scratch: Vec<Complex<f32>>,
  window: [f32; N],
}
impl Analyzer {
  fn new() -> Self {
    let fft = RealFftPlanner::<f32>::new().plan_fft_forward(N);
    Self {
      input: fft.make_input_vec(),
      output: fft.make_output_vec(),
      scratch: fft.make_scratch_vec(),
      fft,
      window: std::array::from_fn(|i| {
        (0.5 - 0.5 * (std::f64::consts::TAU * i as f64 / (N - 1) as f64).cos()) as f32
      }),
    }
  }
  fn analyze(&mut self, samples: &[[f32; N]; 2], cursor: usize) -> [f64; 32] {
    let mut power = [0.0f32; N / 2 + 1];
    for channel in samples {
      for (i, input) in self.input.iter_mut().enumerate() {
        *input = channel[(cursor + i) % N] * self.window[i];
      }
      if self
        .fft
        .process_with_scratch(&mut self.input, &mut self.output, &mut self.scratch)
        .is_err()
      {
        return [0.0; 32];
      }
      for (sum, value) in power.iter_mut().zip(&self.output) {
        *sum += value.norm_sqr() * 0.5;
      }
    }
    std::array::from_fn(|band| {
      let low = 45.0f64 * (16000.0f64 / 45.0).powf(band as f64 / 32.0);
      let high = 45.0f64 * (16000.0f64 / 45.0).powf((band + 1) as f64 / 32.0);
      let start = ((low * N as f64 / RATE).floor() as usize).clamp(1, N / 2);
      let end = ((high * N as f64 / RATE).ceil() as usize).clamp(start + 1, N / 2 + 1);
      let magnitude = power[start..end]
        .iter()
        .copied()
        .fold(0.0f32, f32::max)
        .sqrt() as f64
        * 4.0
        / N as f64;
      if magnitude <= 1e-6 {
        0.0
      } else {
        ((20.0 * magnitude.log10() + 80.0) / 80.0).clamp(0.0, 1.0)
      }
    })
  }
}
struct Spectrum {
  ring: Arc<Mutex<Ring>>,
  analyzer: Mutex<Analyzer>,
}
fn spectrum() -> &'static Spectrum {
  static SPECTRUM: OnceLock<Spectrum> = OnceLock::new();
  SPECTRUM.get_or_init(|| Spectrum {
    ring: Arc::new(Mutex::new(Ring::new())),
    analyzer: Mutex::new(Analyzer::new()),
  })
}
pub fn install() {
  let ring = spectrum().ring.clone();
  librespot_playback::audio_backend::output_tap::set_observer(Some(Arc::new(move |samples| {
    // The real-time output callback must never wait on the UI/FFT.
    if let Ok(mut ring) = ring.try_lock() {
      ring.push(samples);
    }
  })));
}
pub fn bands(active: bool) -> [f64; 32] {
  if !active {
    return [0.0; 32];
  }
  let spectrum = spectrum();
  let snapshot = {
    let Ok(ring) = spectrum.ring.try_lock() else {
      return [0.0; 32];
    };
    if ring.count < N || ring.updated.is_none_or(|time| time.elapsed() > STALE) {
      return [0.0; 32];
    }
    (ring.samples, ring.cursor)
  };
  spectrum
    .analyzer
    .lock()
    .unwrap_or_else(|e| e.into_inner())
    .analyze(&snapshot.0, snapshot.1)
}
#[cfg(test)]
mod tests {
  use super::*;
  #[test]
  fn stereo_frequency_is_measured_even_when_channels_cancel_in_mono() {
    let mut analyzer = Analyzer::new();
    let frequency = 1000.0;
    let left = std::array::from_fn(|i| {
      (std::f64::consts::TAU * frequency * i as f64 / RATE).sin() as f32 * 0.5
    });
    let samples = [left, left.map(|s| -s)];
    let result = analyzer.analyze(&samples, 0);
    let peak = result
      .iter()
      .enumerate()
      .max_by(|a, b| a.1.total_cmp(b.1))
      .unwrap()
      .0;
    let expected = ((frequency / 45.0).ln() / (16000.0f64 / 45.0).ln() * 32.0).floor() as usize;
    assert!(peak.abs_diff(expected) <= 1);
    assert!(result[peak] > 0.8);
    assert_eq!(analyzer.analyze(&[[0.0; N]; 2], 0), [0.0; 32]);
  }
  #[test]
  fn ring_is_bounded_sanitizes_and_wraps() {
    let mut ring = Ring::new();
    ring.push(&vec![f32::NAN; N * 8]);
    assert_eq!(ring.count, N);
    assert!(ring.samples.iter().flatten().all(|s| *s == 0.0));
    ring.push(&[0.5, -0.5]);
    assert_eq!(ring.samples[0][0], 0.5);
    assert_eq!(ring.cursor, 1);
  }
  #[test]
  fn paused_and_stale_output_are_silent() {
    assert_eq!(bands(false), [0.0; 32]);
    let mut ring = spectrum().ring.lock().unwrap();
    ring.push(&vec![0.8; N * 2]);
    ring.updated = Some(Instant::now() - Duration::from_secs(1));
    drop(ring);
    assert_eq!(bands(true), [0.0; 32]);
  }
}
