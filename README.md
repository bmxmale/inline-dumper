# Inline DB Dumper

The `inline-dumper` script is a Go program designed to dump MySQL database tables with various configurable options. It allows users to specify database connection details, choose whether to compress the output, generate checksums, and provide additional `mysqldump` configurations.

## Features

- Connect to a MySQL database and list all tables.
- Dump table data with options for compression and checksums.
- Provide additional `mysqldump` configurations.
- Measure and display the total execution time of the dump process.

## Usage

1. **Clone the repository:**

   ```sh
   git clone git@github.com:bmxmale/inline-dumper.git
   cd inline-dumper
   ```

2. **Build the script:**

   ```sh
   # Build for Linux
   GOOS=linux GOARCH=amd64 go build -o bin/inline-dumper inline-dumper.go
   
   # Build for macOS with Apple M3 processor
   GOOS=darwin GOARCH=arm64 go build -o bin/inline-dumper inline-dumper.go
   ```

3. **Run the script:**

   ```sh
   ./inline-dumper
   ```

4. **Follow the prompts:**

   The script will prompt you for various inputs such as database host, user, password, port, and name. You will also be asked if you want to enable gzip compression, generate checksums, and provide extra `mysqldump` configurations.

## Example

<p align="center" >
<img src="docs/inline-dumper.svg" />
</p> 

## Configuration Options

- **Database Host:** The hostname or IP address of the MySQL server (default: `127.0.0.1`).
- **Database User:** The username to connect to the MySQL server (default: `root`).
- **Database Password:** The password to connect to the MySQL server (default: `root`).
- **Database Port:** The port number on which the MySQL server is listening (default: `3306`).
- **Database Name:** The name of the database to dump (default: `db`).
- **Gzip Compression:** Enable gzip compression for the SQL dump files.
- **Generate Checksums:** Generate a checksums file with MD5 checksums of each dumped file.
- **Extra Configuration:** Provide additional `mysqldump` configurations such as disabling column statistics, skipping table locks, and dumping only table structures.
    - Disable column statistics in mysqldump
    - Dump only table structure without data
    - Skip locking tables during dump

## Acknowledgements

This software was created with the strong support of GitHub Copilot ❤️, an AI-powered code completion tool that helps developers write code faster and with greater accuracy.

With :heart: from Poland 🇵🇱.