package main

import (
	"bufio"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/ssh/terminal"
)

func getTableList(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW FULL TABLES WHERE Table_type = 'BASE TABLE'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	var tableName, tableType string
	for rows.Next() {
		if err := rows.Scan(&tableName, &tableType); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}
	return tables, nil
}

func saveTableListToFile(databaseName string, tables []string) error {
	fileName := fmt.Sprintf("%s.list", databaseName)
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, table := range tables {
		if _, err := file.WriteString(fmt.Sprintf("%s\n", table)); err != nil {
			return err
		}
	}
	return nil
}

func dumpTable(databaseName, tableName, user, password, host string, port int, gzipOutput, generateChecksums, disableColumnStatistics, skipLockTables, noData bool, checksumsFile *os.File) error {
	if _, err := os.Stat(databaseName); os.IsNotExist(err) {
		if err := os.Mkdir(databaseName, 0755); err != nil {
			return err
		}
	}

	dumpFile := filepath.Join(databaseName, fmt.Sprintf("%s.sql", tableName))
	command := fmt.Sprintf("mysqldump --compact --skip-extended-insert --host=%s --port=%d --user=%s %s %s", host, port, user, databaseName, tableName)
	if disableColumnStatistics {
		command += " --column-statistics=0"
	}
	if skipLockTables {
		command += " --skip-lock-tables"
	}
	if noData {
		command += " --no-data"
	}

	if gzipOutput {
		dumpFile += ".gz"
		cmd := exec.Command("sh", "-c", fmt.Sprintf("%s | gzip > %s", command, dumpFile))
		cmd.Env = append(os.Environ(), fmt.Sprintf("MYSQL_PWD=%s", password))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to execute command: %s, output: %s, error: %w", command, output, err)
		}
	} else {
		file, err := os.Create(dumpFile)
		if err != nil {
			return err
		}
		defer file.Close()

		cmd := exec.Command("sh", "-c", command)
		cmd.Stdout = file
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), fmt.Sprintf("MYSQL_PWD=%s", password))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to execute command: %s, error: %w", command, err)
		}
	}

	if generateChecksums {
		checksum, err := fileChecksum(dumpFile)
		if err != nil {
			return err
		}
		if _, err := checksumsFile.WriteString(fmt.Sprintf("%s %s\n", checksum, dumpFile)); err != nil {
			return err
		}
	}

	return nil
}

func fileChecksum(filePath string) (string, error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	buf := make([]byte, 1024*1024) // 1MB buffer
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			break
		}
		if _, err := hash.Write(buf[:n]); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func main() {
	fmt.Println("# Inline DB dumper")
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter database host (default: 127.0.0.1): ")
	host, _ := reader.ReadString('\n')
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}

	fmt.Print("Enter database user (default: root): ")
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)
	if user == "" {
		user = "root"
	}

	fmt.Print("Enter database password (default: root): ")
	bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))
	password := string(bytePassword)
	fmt.Println()
	password = strings.TrimSpace(password)
	if password == "" {
		password = "root"
	}

	fmt.Print("Enter database port (default: 3306): ")
	var port int
	_, err := fmt.Scanf("%d\n", &port)
	if err != nil {
		port = 3306
	}

	fmt.Print("Enter database name (default: db): ")
	databaseName, _ := reader.ReadString('\n')
	databaseName = strings.TrimSpace(databaseName)
	if databaseName == "" {
		databaseName = "db"
	}

	fmt.Print("Enable gzip compression for SQL dump files? (y/n): ")
	gzipResponse, _ := reader.ReadString('\n')
	gzipResponse = strings.TrimSpace(gzipResponse)
	gzipOutput := strings.ToLower(gzipResponse) == "y"

	fmt.Print("Generate checksums file with MD5 checksum of each file dumped? (y/n): ")
	checksumResponse, _ := reader.ReadString('\n')
	checksumResponse = strings.TrimSpace(checksumResponse)
	generateChecksums := strings.ToLower(checksumResponse) == "y"

	fmt.Print("Do you want to provide extra configuration for mysqldump? (y/n): ")
	extraConfigResponse, _ := reader.ReadString('\n')
	extraConfigResponse = strings.TrimSpace(extraConfigResponse)
	extraConfig := strings.ToLower(extraConfigResponse) == "y"

	var disableColumnStatistics, skipLockTables, noData bool
	if extraConfig {
		fmt.Println("# Extra mysqldump options")
		fmt.Print(" - Disable column statistics in mysqldump? (y/n): ")
		columnStatisticsResponse, _ := reader.ReadString('\n')
		columnStatisticsResponse = strings.TrimSpace(columnStatisticsResponse)
		disableColumnStatistics = strings.ToLower(columnStatisticsResponse) == "y"

		fmt.Print(" - Skip locking tables during dump? (y/n): ")
		skipLockTablesResponse, _ := reader.ReadString('\n')
		skipLockTablesResponse = strings.TrimSpace(skipLockTablesResponse)
		skipLockTables = strings.ToLower(skipLockTablesResponse) == "y"

		fmt.Print(" - Dump only table structure without data? (y/n): ")
		noDataResponse, _ := reader.ReadString('\n')
		noDataResponse = strings.TrimSpace(noDataResponse)
		noData = strings.ToLower(noDataResponse) == "y"
	}

	listFileName := fmt.Sprintf("%s.list", databaseName)
	useExistingList := false

	if _, err := os.Stat(listFileName); err == nil {
		fmt.Printf("List file %s already exists. Do you want to use it? (y/n): ", listFileName)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)
		if strings.ToLower(response) == "y" {
			useExistingList = true
		}
	}

	var tables []string
	if useExistingList {
		file, err := os.Open(listFileName)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			tables = append(tables, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		// Inline entries separated by commas
		inlinedTables := strings.Join(tables, ", ")
		fmt.Printf("# Tables: \n%s\n", inlinedTables)
		fmt.Printf("# Total tables: %d\n", len(tables))

		fmt.Print("Do you want to proceed with the dump process based on the selected list? (y/n): ")
		proceed, _ := reader.ReadString('\n')
		proceed = strings.TrimSpace(proceed)
		if strings.ToLower(proceed) != "y" {
			fmt.Println("Dump process aborted.")
			return
		}
	} else {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, databaseName)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		tables, err = getTableList(db)
		if err != nil {
			log.Fatal(err)
		}

		if err := saveTableListToFile(databaseName, tables); err != nil {
			log.Fatal(err)
		}

		// Inline entries separated by commas
		inlinedTables := strings.Join(tables, ", ")
		fmt.Printf("# Tables: \n%s\n", inlinedTables)
		fmt.Printf("# Total tables: %d\n", len(tables))

		fmt.Print("Do you want to proceed with the dump process based on the selected list? (y/n): ")
		proceed, _ := reader.ReadString('\n')
		proceed = strings.TrimSpace(proceed)
		if strings.ToLower(proceed) != "y" {
			fmt.Println("Dump process aborted.")
			return
		}
	}

	var checksumsFile *os.File
	if generateChecksums {
		checksumsFileName := fmt.Sprintf("%s.checksums", databaseName)
		checksumsFile, err = os.Create(checksumsFileName)
		if err != nil {
			log.Fatal(err)
		}
		defer checksumsFile.Close()
	}

	startTime := time.Now()
	for _, table := range tables {
		fmt.Printf(" - %s\n", table)
		if err := dumpTable(databaseName, table, user, password, host, port, gzipOutput, generateChecksums, disableColumnStatistics, skipLockTables, noData, checksumsFile); err != nil {
			log.Fatalf("failed to dump table %s: %v", table, err)
		}
	}
	duration := time.Since(startTime)

	fmt.Println("# Dump process completed.")
	fmt.Printf("# Dump execution time: %s\n", duration)
}
