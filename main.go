package main

import (
	"context"
	"encoding/json"
	"fmt"
	"indigo_auth_svc_api/models"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"

	"cloud.google.com/go/pubsub"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Change * to specific domain for production
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	log.Print("starting server...")
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	mux.HandleFunc("/signup", signupHandler)
	mux.HandleFunc("/sendEmail", checkSMTPServer)
	mux.HandleFunc("/notification", notificationHandler)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	models.PROJECT_ID = os.Getenv("PROJECT_ID")
	if models.PROJECT_ID == "" {
		models.PROJECT_ID = "indigo-437901"
		log.Printf("defaulting to projectID %s", models.PROJECT_ID)
	}
	models.TOPIC_ID = os.Getenv("TOPIC_ID")
	if models.TOPIC_ID == "" {
		models.TOPIC_ID = "indigo-auth-topic"
		log.Printf("defaulting to topicID %s", models.TOPIC_ID)
	}

	models.SUB_ID = os.Getenv("SUB_ID")
	if models.SUB_ID == "" {
		models.SUB_ID = "indigo-auth-sub"
		log.Printf("defaulting to subID %s", models.SUB_ID)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	name := os.Getenv("NAME")
	if name == "" {
		name = "World"
	}
	fmt.Fprintf(w, "Hello %s!\n", name)
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	var message models.Message
	err = json.Unmarshal(body, &message)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	log.Printf("Recieved message: ID=%d, username=%s, email=%s", message.ID, message.USERNAME, message.EMAIL)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Data received successfully"})

	// publish message to Pub/Sub Topic

	err = publishMessage(ctx, models.PROJECT_ID, models.TOPIC_ID, message)
	if err != nil {
		log.Fatal(err)
	}
}

func checkSMTPServer(w http.ResponseWriter, r *http.Request) {
	auth := smtp.PlainAuth(
		"",
		"smtp.paul123@gmail.com",
		"tyhgueeqlvqefslp",
		"smtp.gmail.com",
	)

	// Email details
	from := "smtp.paul123@gmail.com"
	to := []string{"paulsapto@gmail.com"}
	subject := "Subject: Test Email from Go\n"
	body := "This is the email body."

	msg := []byte(subject + "\n" + body)

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, msg)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Email sent successfully!")
	fmt.Fprintf(w, "Email sent successfully!\n")
}

func publishMessage(ctx context.Context, projectID, topicID string, message models.Message) error {
	// Create a context

	// Create a Pub/Sub client
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("pubsub.NewClient: %v", err)
	}

	defer client.Close()

	// Get the topic
	topic := client.Topic(topicID)

	// Serialize the message to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("json.Marshal: %v", err)
	}

	// Publish the message
	result := topic.Publish(ctx, &pubsub.Message{
		Data: jsonData,
	})

	// Wait for the result of the publish
	_, err = result.Get(ctx)
	if err != nil {
		return fmt.Errorf("get: %v", err)

	}

	fmt.Println("Message published successfully.")
	return nil

}
func notificationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recieveMessage(ctx, models.PROJECT_ID, models.SUB_ID)
}

func recieveMessage(ctx context.Context, projectID, subID string) error {
	// ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	defer client.Close()

	sub := client.Subscription(subID)

	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		fmt.Print("Received message....")
		var message models.Message
		err = json.Unmarshal(msg.Data, &message)
		log.Printf("Recieved message: ID=%d, username=%s, email=%s", message.ID, message.USERNAME, message.EMAIL)
		log.Print("sending Notification.....")
		err = sendNotification()
		if err != nil {
			log.Fatalf("Error in Sending notification: %v", err)
		}
		msg.Ack()
	})
	return nil
}

func sendNotification() error {
	auth := smtp.PlainAuth(
		"",
		"smtp.paul123@gmail.com",
		"tyhgueeqlvqefslp",
		"smtp.gmail.com",
	)

	// Email details
	from := "smtp.paul123@gmail.com"
	to := []string{"paulsapto@gmail.com"}
	subject := "Subject: Test Email from Go\n"
	// body := fmt.Sprintf("ID: %d\nUsername: %s\nEmail: %s", message.ID, message.USERNAME, message.EMAIL)
	body := "This is body"

	msg := []byte(subject + "\n" + body)

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, msg)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	fmt.Println("Email sent successfully!")
	return nil
}
