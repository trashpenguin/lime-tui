# Local Lime patch to spotatui-librespot-playback 0.8.6

Original upstream source and license are preserved. This is a local Cargo path override.
Upstream commit: `b6fc24cae85cc7af943f52cfd5250f1002ebbdff`, `playback/`,
https://github.com/LargeModGames/spotatui-librespot. The root MIT LICENSE is included.

The optional `pcm-observer` feature wraps the rodio source iterator with a bounded
1024-sample observer. It publishes samples when the output mixer consumes them,
after the decoder queue, so Lime can analyze its own outgoing audio. It does not
capture a microphone or system audio, change samples, or save audio. No lock is
taken per sample. Observers must not block or panic. The unit test verifies that
enqueuing publishes nothing and consumption forwards the exact samples in order.
