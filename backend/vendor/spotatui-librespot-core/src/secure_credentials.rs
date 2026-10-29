//! Lime Windows credential persistence. Plaintext is never written by this module.
//! DPAPI is scoped to the current Windows user, with a fixed application purpose.
use crate::authentication::Credentials;
use std::{
  fs::{self, File, OpenOptions},
  io::{self, Read, Write},
  os::windows::{ffi::OsStrExt, fs::OpenOptionsExt},
  path::Path,
  sync::atomic::{AtomicU64, Ordering},
};
use windows_sys::Win32::{
  Foundation::LocalFree,
  Security::Cryptography::{
    CRYPT_INTEGER_BLOB, CRYPTPROTECT_UI_FORBIDDEN, CryptProtectData, CryptUnprotectData,
  },
  Storage::FileSystem::{MOVEFILE_REPLACE_EXISTING, MOVEFILE_WRITE_THROUGH, MoveFileExW},
};
use zeroize::Zeroizing;

const MAGIC: &[u8] = b"LIMEDP01";
const PURPOSE: &[u8] = b"Lime TUI Spotify credentials v1";
const LIMIT: u64 = 64 * 1024;

fn crypt(input: &[u8], decrypt: bool) -> io::Result<Zeroizing<Vec<u8>>> {
  let input = CRYPT_INTEGER_BLOB {
    cbData: input
      .len()
      .try_into()
      .map_err(|_| io::Error::other("Credential size exceeds DPAPI limit"))?,
    pbData: input.as_ptr() as *mut u8,
  };
  let entropy = CRYPT_INTEGER_BLOB {
    cbData: PURPOSE.len() as u32,
    pbData: PURPOSE.as_ptr() as *mut u8,
  };
  let mut output = CRYPT_INTEGER_BLOB {
    cbData: 0,
    pbData: std::ptr::null_mut(),
  };
  // SAFETY: all slices remain alive for the call; Windows allocates output with LocalAlloc.
  let ok = unsafe {
    if decrypt {
      CryptUnprotectData(
        &input,
        std::ptr::null_mut(),
        &entropy,
        std::ptr::null(),
        std::ptr::null(),
        CRYPTPROTECT_UI_FORBIDDEN,
        &mut output,
      )
    } else {
      CryptProtectData(
        &input,
        std::ptr::null(),
        &entropy,
        std::ptr::null(),
        std::ptr::null(),
        CRYPTPROTECT_UI_FORBIDDEN,
        &mut output,
      )
    }
  };
  if ok == 0 {
    return Err(io::Error::last_os_error());
  }
  if output.pbData.is_null() {
    return Err(io::Error::other("DPAPI returned no credential data"));
  }
  // SAFETY: DPAPI returned cbData valid bytes. Erase its buffer before freeing it.
  let bytes = unsafe { std::slice::from_raw_parts_mut(output.pbData, output.cbData as usize) };
  let result = Zeroizing::new(bytes.to_vec());
  zeroize::Zeroize::zeroize(bytes);
  unsafe {
    LocalFree(output.pbData as *mut _);
  }
  Ok(result)
}

fn bounded_read(path: &Path) -> io::Result<Zeroizing<Vec<u8>>> {
  let metadata = fs::symlink_metadata(path)?;
  if !metadata.is_file() || metadata.file_type().is_symlink() || metadata.len() > LIMIT {
    return Err(io::Error::new(
      io::ErrorKind::InvalidData,
      "Invalid credential cache file",
    ));
  }
  let mut bytes = Zeroizing::new(Vec::new());
  File::open(path)?.take(LIMIT + 1).read_to_end(&mut bytes)?;
  if bytes.len() as u64 > LIMIT {
    return Err(io::Error::new(
      io::ErrorKind::InvalidData,
      "Credential cache is too large",
    ));
  }
  Ok(bytes)
}

fn lock(location: &Path) -> io::Result<File> {
  let path = location.with_file_name("credentials.lock");
  for attempt in 0..100 {
    match OpenOptions::new()
      .read(true)
      .write(true)
      .create(true)
      .truncate(false)
      .share_mode(0)
      .open(&path)
    {
      Ok(file) => return Ok(file),
      Err(e) if e.raw_os_error() == Some(32) && attempt < 99 => {
        std::thread::sleep(std::time::Duration::from_millis(10))
      }
      Err(e) => return Err(e),
    }
  }
  Err(io::Error::other("Credential cache is busy"))
}

fn protected_read(path: &Path) -> io::Result<Credentials> {
  let bytes = bounded_read(path)?;
  let blob = bytes.strip_prefix(MAGIC).ok_or_else(|| {
    io::Error::new(
      io::ErrorKind::InvalidData,
      "Unsupported protected credential format",
    )
  })?;
  let plaintext = crypt(blob, true)?;
  serde_json::from_slice(&plaintext)
    .map_err(|_| io::Error::new(io::ErrorKind::InvalidData, "Invalid protected credentials"))
}

fn wide(path: &Path) -> io::Result<Vec<u16>> {
  let mut value: Vec<_> = path.as_os_str().encode_wide().collect();
  if value.contains(&0) {
    return Err(io::Error::new(
      io::ErrorKind::InvalidInput,
      "Invalid credential path",
    ));
  }
  value.push(0);
  Ok(value)
}

fn protected_write(path: &Path, credentials: &Credentials) -> io::Result<()> {
  let plaintext = Zeroizing::new(
    serde_json::to_vec(credentials)
      .map_err(|_| io::Error::other("Could not serialize credentials"))?,
  );
  let ciphertext = crypt(&plaintext, false)?;
  static SEQUENCE: AtomicU64 = AtomicU64::new(0);
  let temp = path.with_file_name(format!(
    "credentials.dpapi.{}.{}.tmp",
    std::process::id(),
    SEQUENCE.fetch_add(1, Ordering::Relaxed)
  ));
  let result = (|| {
    let mut file = OpenOptions::new()
      .write(true)
      .create_new(true)
      .open(&temp)?;
    file.write_all(MAGIC)?;
    file.write_all(&ciphertext)?;
    file.sync_all()?;
    drop(file);
    let from = wide(&temp)?;
    let to = wide(path)?;
    // SAFETY: both UTF-16 strings are NUL terminated and live through the call.
    if unsafe {
      MoveFileExW(
        from.as_ptr(),
        to.as_ptr(),
        MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH,
      )
    } == 0
    {
      return Err(io::Error::last_os_error());
    }
    if protected_read(path)? != *credentials {
      return Err(io::Error::other("Protected credential verification failed"));
    }
    Ok(())
  })();
  if result.is_err() {
    let _ = fs::remove_file(temp);
  }
  result
}

fn remove_legacy(path: &Path) -> io::Result<()> {
  match fs::remove_file(path) {
    Ok(()) => Ok(()),
    Err(e) if e.kind() == io::ErrorKind::NotFound => Ok(()),
    Err(e) => Err(e),
  }
}

pub(crate) fn load(legacy: &Path) -> io::Result<Credentials> {
  let _guard = lock(legacy)?;
  let protected = legacy.with_file_name("credentials.dpapi");
  match protected_read(&protected) {
    Ok(credentials) => {
      remove_legacy(legacy)?;
      return Ok(credentials);
    }
    Err(e) if e.kind() == io::ErrorKind::NotFound => {}
    Err(e) => return Err(e), // Never fall back to plaintext after corruption/decryption failure.
  }
  let bytes = bounded_read(legacy)?;
  let credentials: Credentials = serde_json::from_slice(&bytes)
    .map_err(|_| io::Error::new(io::ErrorKind::InvalidData, "Invalid legacy credentials"))?;
  protected_write(&protected, &credentials)?;
  remove_legacy(legacy)?;
  Ok(credentials)
}

pub(crate) fn save(legacy: &Path, credentials: &Credentials) -> io::Result<()> {
  let _guard = lock(legacy)?;
  protected_write(&legacy.with_file_name("credentials.dpapi"), credentials)?;
  remove_legacy(legacy)
}
