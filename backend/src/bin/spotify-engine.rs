#[tokio::main(flavor = "multi_thread", worker_threads = 2)]
async fn main() {
  if let Err(error) = spotatui::run_pomme().await {
    eprintln!("Spotify engine: {error:#}");
    std::process::exit(1);
  }
}
