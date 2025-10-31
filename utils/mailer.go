package utils

import (
	"fmt"
	"log"
	"net/smtp"
)

func SendLeakAlertMail(smtpHost, smtpPort, smtpUser, smtpPass, toEmail, keyName, keyValue, repoURL string) error {
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	subject := "Secret Key Leak Detected"
	body := fmt.Sprintf(
		"Your key %s with value %s is being leaked in the repository %s.",
		keyName, keyValue, repoURL,
	)

	msg := []byte("Subject: " + subject + "\r\n" +
		"From: " + smtpUser + "\r\n" +
		"To: " + toEmail + "\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n" +
		body + "\r\n")

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	err := smtp.SendMail(addr, auth, smtpUser, []string{toEmail}, msg)
	if err != nil {
		log.Printf("Failed to send email to %s: %v", toEmail, err)
		return err
	}

	log.Printf("Email sent successfully to %s (key: %s)", toEmail, keyName)
	return nil
}
