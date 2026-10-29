//! Offline Windows security checks use synthetic credentials in disposable directories.
use librespot_core::{authentication::Credentials, cache::Cache};

fn fixture() -> Credentials {
  Credentials::with_access_token("lime-test-secret-never-a-real-token")
}
fn cache(path: &std::path::Path) -> Cache {
  Cache::new(Some(path), None, None, None).unwrap()
}

#[test]
fn protected_cache_roundtrip_and_atomic_replacement() {
  let dir = tempfile::tempdir().unwrap();
  let cache = cache(dir.path());
  cache.save_credentials(&fixture());
  let bytes = std::fs::read(dir.path().join("credentials.dpapi")).unwrap();
  assert!(bytes.starts_with(b"LIMEDP01"));
  assert!(!bytes.windows(16).any(|s| s == b"lime-test-secret"));
  assert!(!dir.path().join("credentials.json").exists());
  assert_eq!(cache.credentials(), Some(fixture()));
  let updated = Credentials::with_access_token("replacement-fixture");
  cache.save_credentials(&updated);
  assert_eq!(cache.credentials(), Some(updated));
  assert!(!std::fs::read_dir(dir.path()).unwrap().any(|f| f
    .unwrap()
    .path()
    .extension()
    .is_some_and(|s| s == "tmp")));
}

#[test]
fn legacy_login_is_verified_and_migrated_without_plaintext_backup() {
  let dir = tempfile::tempdir().unwrap();
  let legacy = dir.path().join("credentials.json");
  std::fs::write(&legacy, serde_json::to_vec(&fixture()).unwrap()).unwrap();
  assert_eq!(cache(dir.path()).credentials(), Some(fixture()));
  assert!(!legacy.exists());
  assert!(dir.path().join("credentials.dpapi").exists());
}

#[test]
fn corrupted_dpapi_is_rejected_without_plaintext_fallback() {
  let dir = tempfile::tempdir().unwrap();
  let cache = cache(dir.path());
  cache.save_credentials(&fixture());
  let path = dir.path().join("credentials.dpapi");
  let mut bytes = std::fs::read(&path).unwrap();
  let last = bytes.len() - 1;
  bytes[last] ^= 0x40;
  std::fs::write(&path, bytes).unwrap();
  std::fs::write(
    dir.path().join("credentials.json"),
    serde_json::to_vec(&fixture()).unwrap(),
  )
  .unwrap();
  assert!(cache.credentials().is_none());
}

#[test]
fn failed_migration_preserves_only_recoverable_legacy_login() {
  let dir = tempfile::tempdir().unwrap();
  let legacy = dir.path().join("credentials.json");
  let data = serde_json::to_vec(&fixture()).unwrap();
  std::fs::write(&legacy, &data).unwrap();
  std::fs::create_dir(dir.path().join("credentials.dpapi")).unwrap();
  assert!(cache(dir.path()).credentials().is_none());
  assert_eq!(std::fs::read(legacy).unwrap(), data);
}

#[test]
fn concurrent_saves_never_leave_partial_or_plaintext_credentials() {
  let dir = tempfile::tempdir().unwrap();
  let cache = cache(dir.path());
  let threads: Vec<_> = (0..8)
    .map(|i| {
      let cache = cache.clone();
      std::thread::spawn(move || {
        cache.save_credentials(&Credentials::with_access_token(format!("fixture-{i}")))
      })
    })
    .collect();
  for t in threads {
    t.join().unwrap();
  }
  let saved = cache.credentials().unwrap();
  assert!((0..8).any(|i| saved == Credentials::with_access_token(format!("fixture-{i}"))));
  assert!(!dir.path().join("credentials.json").exists());
}

#[test]
fn sign_out_removes_both_protected_and_legacy_formats() {
  let dir = tempfile::tempdir().unwrap();
  let cache = cache(dir.path());
  cache.save_credentials(&fixture());
  std::fs::write(
    dir.path().join("credentials.json"),
    serde_json::to_vec(&fixture()).unwrap(),
  )
  .unwrap();
  crate::infra::spotify::credentials::clear_cached_streaming_credentials(&Some(
    dir.path().to_path_buf(),
  ));
  assert!(cache.credentials().is_none());
  assert!(!dir.path().join("credentials.json").exists());
  assert!(!dir.path().join("credentials.dpapi").exists());
}

#[test]
fn oversized_cache_is_rejected() {
  let dir = tempfile::tempdir().unwrap();
  std::fs::write(dir.path().join("credentials.json"), vec![b'x'; 65537]).unwrap();
  assert!(cache(dir.path()).credentials().is_none());
}

#[test]
fn product_info_parser_is_bounded_and_rejects_malformed_xml() {
  use librespot_core::session::parse_product_info;
  let attrs = parse_product_info(
    "<products><product><type>premium</type><country>US</country></product></products>",
  )
  .unwrap();
  assert_eq!(attrs.get("type").map(String::as_str), Some("premium"));
  assert!(parse_product_info("<products><type>premium</products>").is_err());
  assert!(parse_product_info(&"x".repeat(65537)).is_err());
}
