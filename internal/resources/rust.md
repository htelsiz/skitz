# Rust

`cargo build` build project ^run
`cargo build --release` build release ^run
`cargo run` build and run ^run
`cargo run -- {{args}}` run with arguments ^run:args
`cargo test` run all tests ^run
`cargo test {{name}}` run matching tests ^run:name
`cargo test -- --nocapture` run tests with output ^run
`cargo bench` run benchmarks ^run
`cargo check` type-check without building ^run
`cargo clippy` run linter ^run
`cargo fmt` format code ^run
`cargo fmt --check` check formatting ^run
`cargo doc --open` generate and open docs ^run
`cargo add {{crate}}` add dependency ^run:crate
`cargo remove {{crate}}` remove dependency ^run:crate
`cargo update` update dependencies ^run
`cargo tree` show dependency tree ^run
`cargo init {{name}}` create new project ^run:name
`cargo new {{name}}` create new package ^run:name
`cargo clean` remove build artifacts ^run
`cargo install {{crate}}` install binary ^run:crate
`cargo publish` publish to crates.io ^run
`cargo audit` check for vulnerabilities ^run
`rustup update` update toolchain ^run
`rustup show` show installed toolchains ^run
`rustup target add {{target}}` add compile target ^run:target
