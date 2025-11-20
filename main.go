package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/gomail.v2"
	//"github.com/joho/godotenv"
)

func generatePassword(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func main() {

	user := os.Getenv("DBUSER")
	dbpass := os.Getenv("DBPASSWORD")
	host := os.Getenv("DBHOST")
	dbname := os.Getenv("DBNAME")
	port := os.Getenv("DBPORT")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, dbpass, host, port, dbname)

	log.Println("DSN:", dsn)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("DB connect error:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("DB ping error:", err)
	}

	log.Println("DB connection OK")

	// start every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// start immediately
	runJob(db)

	for range ticker.C {
		runJob(db)
	}
}

func runJob(db *sql.DB) {

	_, err := db.Exec("DELETE FROM entry_password WHERE is_active = TRUE")
	if err != nil {
		log.Println("failed to delete old password:", err)
		return
	}

	entrypass, err := generatePassword(12)
	if err != nil {
		log.Printf("Error generating password: %v", err)
		return
	}

	_, err = db.Exec("INSERT INTO entry_password (password, is_active, created_at, updated_at) VALUES (?, TRUE, NOW(), NOW())", entrypass)
	if err != nil {
		log.Println("insert password failed:", err)
		return
	}

	rows, err := db.Query("SELECT email FROM users_customuser")
	if err != nil {
		log.Println("query emails failed:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			log.Printf("row scan failed: %v", err)
			continue
		}

		passsmtp := os.Getenv("SMTP_PASS")
		emailsmtp := os.Getenv("SMTP_EMAIL")

		m := gomail.NewMessage()
		m.SetHeader("From", emailsmtp)
		m.SetHeader("To", email)
		m.SetHeader("Subject", "Hello!")
		m.SetBody("text/plain", "Hello from notmason!")
		m.SetAddressHeader("Cc", emailsmtp, "Vandrey")

		m.SetHeader("Subject", "Daily Password")
		m.SetBody("text/plain", fmt.Sprintf("Your password: %s", entrypass))
		m.AddAlternative("text/html", fmt.Sprintf("<b>Your password:</b> %s", entrypass))

		d := gomail.NewDialer("smtp.gmail.com", 587, emailsmtp, passsmtp)

		if err := d.DialAndSend(m); err != nil {
			log.Printf("failed to send to %s: %v", email, err)
			continue
		} else {
			fmt.Printf("Email sent to %s\n", email)
		}

	}
}
