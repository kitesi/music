use anyhow::{anyhow, Context, Result};
use clap::{Parser, Subcommand};
use home;
use regex::Regex;
use walkdir::WalkDir;

use std::{fs, path::PathBuf, process};

#[derive(Parser)]
#[clap(name = "music", author)]
struct Cli {
    #[clap(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// install music with youtube-dl
    Install {
        /// the id of the youtube video, or a full url
        id: String,
        /// the folder to save to
        folder: String,
    },
    // #[clap(external_subcommand)]
    // External(Vec<String>)
    Play {
        #[clap(long, short)]
        dry_run: bool,
        #[clap(long, short = 'p')]
        dry_paths: bool,
        #[clap(long, short)]
        new: bool,
        #[clap(long)]
        play_new_first: bool,
        #[clap(long)]
        delete_old_first: bool,
        #[clap(long, short)]
        limit: Option<i32>,
        #[clap(long)]
        persist: bool,
        #[clap(long, default_value = "vlc")]
        vlc_path: String,
        #[clap(long, default_value = "modified")]
        /// values: a, m, c
        sort_type: String,
        terms: Vec<String>,
    },
}

fn does_song_pass(terms: &Vec<String>, music_path: &PathBuf, song_path: &PathBuf) -> bool {
    if terms.len() == 0 {
        return true;
    }

    let mut passed_one_term = false;

    for term in terms {
        let mut term = term.clone().to_lowercase();
        let is_exclusion = term.starts_with('!');

        if is_exclusion {
            term.remove(0);
        }

        let required_section_seperator = Regex::new(r"#\s*").expect("Invalid regex");
        let optional_section_seperator = Regex::new(r",\s*").expect("Invalid regex");

        let mut required_sections = required_section_seperator.split(&term);
        let song_path = song_path
            .to_str()
            .expect(&format!(
                "Non UTF-8 Characters for file: {}",
                song_path.display()
            ))
            .to_ascii_lowercase();

        let song_path = song_path.replace(music_path.to_str().unwrap(), "");

        if required_sections.all(|required_section| {
            optional_section_seperator
                .split(required_section)
                .any(|optional_section| song_path.contains(optional_section))
        }) {
            if is_exclusion {
                return false;
            }

            passed_one_term = true;
        }
    }

    passed_one_term
}

fn get_songs_by_terms(
    music_path: &PathBuf,
    terms: &Vec<String>,
    limit: usize,
) -> Result<Vec<String>> {
    let mut chosen_songs: Vec<String> = vec![];

    for entry in WalkDir::new(music_path).into_iter().filter_map(|e| e.ok()) {
        if entry.file_type().is_dir() {
            continue;
        }

        if does_song_pass(terms, music_path, &entry.path().to_path_buf()) {
            if chosen_songs.len() == limit {
                return Ok(chosen_songs);
            }

            chosen_songs.push(entry.path().to_str().unwrap().to_owned());
        }
    }

    Ok(chosen_songs)
}

fn play_vlc<T: std::convert::AsRef<std::ffi::OsStr>>(
    music_path: &PathBuf,
    vlc_path: &str,
    vlc_args: &[T],
    dry_run: bool,
    persist: bool,
) -> Result<()> {
    if dry_run {
        return Ok(());
    }

    let mut command = process::Command::new(vlc_path);

    command
        .current_dir(music_path)
        .args(vlc_args)
        .stderr(process::Stdio::null());

    if persist {
        command
            .output()
            .with_context(|| "Error trying to play music")?;
    } else {
        command
            .stdout(process::Stdio::null())
            .spawn()
            .with_context(|| "Error trying to play music")?;
    }

    Ok(())
}

fn main() -> Result<()> {
    let args = Cli::parse();

    let mut music_path = home::home_dir().expect("Could not retrive the home directory");
    music_path.push("Music");

    match &args.command {
        Commands::Install { id, folder } => {
            println!("{}, {}", id, folder);
        }
        Commands::Play {
            delete_old_first,
            dry_paths,
            dry_run,
            limit,
            new,
            play_new_first,
            persist,
            sort_type,
            terms,
            vlc_path,
        } => {
            let has_limit = limit.is_some();
            let limit: usize = limit.unwrap_or(500).try_into().unwrap();

            if !has_limit && terms.len() == 0 && !dry_paths && !play_new_first && !new {
                println!("Playing all songs");

                play_vlc(
                    &music_path,
                    &vlc_path,
                    &["--recursive=expand", music_path.to_str().unwrap()],
                    *dry_run,
                    *persist,
                )?;

                return Ok(());
            }

            let mut songs = get_songs_by_terms(&music_path, &terms, limit)?;

            if songs.len() == 0 {
                return Err(anyhow!("Didn't match anything"));
            }

            let sort_by_new = |a: &String, b: &String| {
                let song_a_stats = fs::metadata(a).expect(&format!("Error trying to read '{}'", a));
                let song_b_stats = fs::metadata(b).expect(&format!("Error trying to read '{}'", b));

                let times = match sort_type.as_str() {
                    "a" | "accessed" => (song_a_stats.accessed(), song_b_stats.accessed()),
                    "c" | "created" => (song_a_stats.created(), song_b_stats.created()),
                    "m" | "modified" | _ => (song_a_stats.modified(), song_b_stats.modified()),
                };

                times.0.unwrap().partial_cmp(&times.1.unwrap()).unwrap()
            };

            if *new || *delete_old_first {
                songs.sort_by(sort_by_new);
            }

            if songs.len() > limit {
                songs.truncate(limit);
            }

            // !new && !delete_old_first to make sure we don't uselessly sort again
            if *play_new_first && !new && !delete_old_first {
                songs.sort_by(sort_by_new);
            }

            if *dry_paths {
                for song in songs {
                    println!("{}", song);
                }

                return Ok(());
            }

            if !has_limit && terms.len() == 0 {
                println!("Playing all songs [{}]", songs.len());
            } else {
                println!("Playing: [{}]", songs.len());

                let songs_clean_path: Vec<_> = songs
                    .iter()
                    .map(|song| {
                        let mut song = song.as_str().replace(music_path.to_str().unwrap(), "");
                        song.remove(0);
                        song
                    })
                    .collect();

                for song in &songs_clean_path {
                    println!("- {}", song);
                }
            }

            if *new || *play_new_first {
                songs.push("--no-random".to_owned());
            }

            play_vlc(&music_path, &vlc_path, &songs, *dry_run, *persist)?;
        }
    };

    Ok(())
}
