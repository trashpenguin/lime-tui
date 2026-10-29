//! Optional bounded observer at source consumption, after the decoder queue.
use rodio::Source;
use std::sync::{Arc, OnceLock, RwLock};
use std::time::Duration;

pub type Observer = Arc<dyn Fn(&[f32]) + Send + Sync>;
fn observer() -> &'static RwLock<Option<Observer>> {
  static OBSERVER: OnceLock<RwLock<Option<Observer>>> = OnceLock::new();
  OBSERVER.get_or_init(|| RwLock::new(None))
}
/// The observer must not block, panic, or retain unbounded samples.
pub fn set_observer(value: Option<Observer>) {
  *observer().write().unwrap_or_else(|e| e.into_inner()) = value;
}
pub struct OutputTap<S> {
  source: S,
  observer: Option<Observer>,
  block: [f32; 1024],
  used: usize,
}
impl<S> OutputTap<S> {
  pub fn new(source: S) -> Self {
    Self {
      source,
      observer: observer().read().unwrap_or_else(|e| e.into_inner()).clone(),
      block: [0.0; 1024],
      used: 0,
    }
  }
  fn flush(&mut self) {
    if self.used != 0 {
      if let Some(observer) = &self.observer {
        observer(&self.block[..self.used]);
      }
      self.used = 0;
    }
  }
}
impl<S: Source> Iterator for OutputTap<S> {
  type Item = f32;
  fn next(&mut self) -> Option<f32> {
    let sample = self.source.next();
    if self.observer.is_some() {
      if let Some(value) = sample {
        self.block[self.used] = value;
        self.used += 1;
        if self.used == self.block.len() {
          self.flush();
        }
      } else {
        self.flush();
      }
    }
    sample
  }
  fn size_hint(&self) -> (usize, Option<usize>) {
    self.source.size_hint()
  }
}
impl<S: Source> Source for OutputTap<S> {
  fn current_span_len(&self) -> Option<usize> {
    self.source.current_span_len()
  }
  fn channels(&self) -> rodio::ChannelCount {
    self.source.channels()
  }
  fn sample_rate(&self) -> rodio::SampleRate {
    self.source.sample_rate()
  }
  fn total_duration(&self) -> Option<Duration> {
    self.source.total_duration()
  }
}

#[cfg(test)]
mod tests {
  use super::*;
  use std::sync::Mutex;
  #[test]
  fn enqueue_does_not_publish_only_consumed_samples_do() {
    let observed = Arc::new(Mutex::new(Vec::new()));
    let target = observed.clone();
    set_observer(Some(Arc::new(move |samples| {
      target.lock().unwrap().extend_from_slice(samples)
    })));
    let values = vec![0.25f32; 1536];
    let source = rodio::buffer::SamplesBuffer::new(2, 44100, values.clone());
    let mut tap = OutputTap::new(source);
    assert!(observed.lock().unwrap().is_empty());
    for _ in 0..1024 {
      assert_eq!(tap.next(), Some(0.25));
    }
    assert_eq!(observed.lock().unwrap().len(), 1024);
    let rest: Vec<_> = tap.collect();
    assert_eq!(rest.len(), 512);
    assert_eq!(*observed.lock().unwrap(), values);
    set_observer(None);
  }
}
