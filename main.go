package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"log"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Erzeuge Strukt mit Feldern für Id, Methode, URL und Zeitpunkt zu dem die Anfrage erstellt wurde.
// Zusätzlich wird die Umwandlung in das json-Format definiert
type Request struct {
	ID          string            `json:"id"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Timestamp   time.Time         `json:"timestamp"`
	RemoteAddr  string            `json:"remote_addr"`
	UserAgent   string            `json:"user_agent"`
	ContentType string            `json:"content_type"`
	BodyParams  map[string]string `json:"body_params"`
	LinkToFile  string            `json:"link_to_file"`
}

// Slice von Requests anlegen
var requests []Request

// Lese alle Requests aus der Datei /requests und Speichere sie nach Erstelldatum sortiert in die Slice requests
func restoreRequests() {
	entries, err := os.ReadDir("./requests")
	if err != nil {
		log.Fatal("Fehler beim Lesen der Dateien:", err)
	}

	var reqs []Request

	for _, entry := range entries {
		data, err := os.ReadFile(fmt.Sprintf("./requests/%s", entry.Name()))
		if err != nil {
			log.Println("Fehler beim Lesen der Datei:", err)
			continue
		}

		var req Request
		if err := json.Unmarshal(data, &req); err != nil {
			log.Println("Fehler beim Entmarshalling des Requests:", err)
			continue
		}
		reqs = append(reqs, req)
	}

	// Sortiere die Anfragen nach Erstelldatum
	sort.Slice(reqs, func(i, j int) bool {
		return reqs[j].Timestamp.Before(reqs[i].Timestamp)
	})

	requests = reqs
}

// Erhält einen gin.Context und wandelt diesen direkt in eine Request-Struct um
func parseRequest(c *gin.Context) Request {
	// Initialisiere eine leere Map, um die Body-Parameter zu speichern
	bodyParams := make(map[string]string)

	// Überprüfe, ob die Anfrage eine POST- oder PUT-Anfrage ist
	if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
		// Ermittle den Content-Type Header der Anfrage
		contentType := c.GetHeader("Content-Type")

		// Überprüfe, ob es sich um einen "multipart/form-data" Content-Type handelt
		if strings.HasPrefix(contentType, "multipart/form-data") {
			// Parsen des Multipart-Formulars
			if err := c.Request.ParseMultipartForm(0); err != nil {
				log.Println("Fehler beim Parsen des Multipart-Formulars:", err)
			} else {
				// Durchlaufe die PostForm, um die Parameter zu extrahieren und in bodyParams zu speichern
				for key, values := range c.Request.PostForm {
					bodyParams[key] = values[0]
				}
			}
		} else if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
			// Parsen des Formulars
			if err := c.Request.ParseForm(); err != nil {
				log.Println("Fehler beim Parsen des Formulars:", err)
			} else {
				// Durchlaufe die PostForm, um die Parameter zu extrahieren und in bodyParams zu speichern
				for key, values := range c.Request.PostForm {
					bodyParams[key] = values[0]
				}
			}
		}
		// Wenn keine Parameter bestimmt werden können und der Body eine Länge > 0 hat
		if len(bodyParams) == 0 && c.Request.ContentLength > 0 {
			fileContent := c.Request.Body
			// Lies den Body-Inhalt aus
			bodyContent, err := io.ReadAll(fileContent)
			if err != nil {
				log.Println("Fehler beim Lesen des Request-Body:", err)
			} else {
				// Generiere einen Dateinamen
				// Erkenne die Dateiendung aus dem Content-Type
				extension, err2 := mime.ExtensionsByType(contentType)
				if err2 != nil {
					fmt.Println(err2)
				}

				filename := fmt.Sprintf("%s%s", generateRandomString(6), extension[0])

				// Speichere den Body-Inhalt in der "static-files"-Datei
				err := os.WriteFile(filepath.Join("static-files", filename), bodyContent, 0644)
				if err != nil {
					log.Println("Fehler beim Speichern des Body-Inhalts:", err)
				}

				// Erstelle und gib eine Request-Struktur mit den extrahierten Informationen zurück,
				// inklusive dem Dateilink
				return Request{
					ID:          uuid.New().String(),
					Method:      c.Request.Method,
					URL:         c.Request.URL.String(),
					Timestamp:   time.Now(),
					RemoteAddr:  c.ClientIP(),
					UserAgent:   c.Request.UserAgent(),
					ContentType: c.ContentType(),
					BodyParams:  bodyParams,
					LinkToFile:  fmt.Sprintf("http://localhost:8080/static/%s", filename), // Setze den Dateilink
				}
			}
		}
	}

	// Erstelle und gib eine Request-Struktur mit den extrahierten Informationen zurück
	return Request{
		ID:          uuid.New().String(),
		Method:      c.Request.Method,
		URL:         c.Request.URL.String(),
		Timestamp:   time.Now(),
		RemoteAddr:  c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		ContentType: c.ContentType(),
		BodyParams:  bodyParams,
	}
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func handleTestRequest(forwardReqs chan<- Request) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Gebe "Hello World" unter dem Statuscode 200 aus
		c.String(200, "Hello World\n")
		c.String(200, "Hello Universe")

		// Rufe die saveRequest Methode mit einem in eine Struct umgewandelten Request
		req := parseRequest(c)
		forwardReqs <- req
		saveRequest(req)
	}

}

// Gebe die Anzahl der Anfragen aus
func requestCounter(c *gin.Context) {
	fmt.Printf("Anzahl der Requests: %d\n", len(requests))
	c.String(200, fmt.Sprintf("Anzahl der Requests: %d", len(requests)))
}

// Speichern einer Request-Struct an den Anfang einer Slice sowie in eine Datei
func saveRequest(r Request) {
	// Füge die Request-Struct der Slice hinzu
	requests = append([]Request{r}, requests...)

	// Speichere die Request-Struct in eine Datei
	saveToFile(r)
}

// Speichere die Request-Struct in eine Datei
func saveToFile(r Request) {
	// Formatieren des JSON-Strings mit Zeilenumbrüchen für bessere Lesbarkeit
	data, err := json.MarshalIndent(r, "\n", "    ")
	if err != nil {
		log.Println("Fehler beim Marshalling des Requests:", err)
		return
	}

	// Speichern der Datei
	err = os.WriteFile(fmt.Sprintf("./requests/%s.json", r.ID), data, 0644)
	if err != nil {
		log.Println("Fehler beim Schreiben der Datei:", err)
	}
}

// Zeigt eine Liste von Requests basierend auf dem Query-Parameter "p" an.
func viewRequests(c *gin.Context) {
	// Query-Parameter 'p' auslesen
	pageStr := c.DefaultQuery("p", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		c.String(400, "Ungültige Seitennummer")
		return
	}

	// Anzahl der Requests pro Seite und Start-/Endindex berechnen
	requestsPerPage := 10
	startIndex := (page - 1) * requestsPerPage
	endIndex := startIndex + requestsPerPage

	// Holen der gewünschten Anzahl von Requests aus der Slice
	currentRequestSlice := getSliceElements(requests, startIndex, endIndex)
	c.Header("Access-Control-Allow-Origin", "*")
	c.JSON(200, currentRequestSlice)
}

func getSliceElements(slice []Request, start, end int) []Request {
	if start < 0 || start > end || start >= len(slice) {
		return []Request{}
	}

	if end > len(slice) {
		end = len(slice)
	}

	return slice[start:end]
}

// Funktion zum Anlegen des Verzeichnisses
// Legt die Datei requests an: Home/GolandProject/awesomeProjects
func createRequestsDirectory() error {
	return os.MkdirAll("./requests", 0755) // 0755 sind die Berechtigungen
}

// Zusatzaufgabe 2: Echtzeitkommunikation mit dem Browser (Websockets oder SSE)

// Diese Funktion akzeptiert eine Request-Struktur und sendet sie an alle registrierten SSE-Clients.
func SendToAllClients(req Request) {
	// Wandel das Request in json um
	var data, err = json.Marshal(req)
	if err == nil {
		// Fülle den allClients chan mit dem String des Requests
		for clientId, client := range SSEClients {
			fmt.Printf("Sending to client %s", clientId)
			dataString := fmt.Sprintf("event: message\ndata: %s\n\n", string(data))
			client <- string(dataString)
		}
	}
}

// Musste global angelegt werden
var SSEClients = make(map[string]chan string)

// Wird mit einer bestimmten Anzahl an Requests aufgerufen und sendet diese an alle Klienten aus SSEClients
func reciver(requests <-chan Request) {
	for req := range requests {
		// Fülle den SSEClients chan mit dem String des Requests
		SendToAllClients(req)
	}
}

func SSEHandler(requestsChan <-chan Request) gin.HandlerFunc {

	// Fülle die SSEClients für alle Requests
	go reciver(requestsChan)

	return func(c *gin.Context) {
		clientChannel := make(chan string)
		clientId := generateRandomString(50)
		SSEClients[clientId] = clientChannel
		fmt.Println("Client connected: ", clientId)
		// Set the response headers for SSE
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")

		flusher := c.Writer.(http.Flusher)
		flusher.Flush()

		closeNotifier, ok := c.Writer.(http.CloseNotifier)
		if !ok {
			http.Error(c.Writer, "Server-Sent Events are not supported.", http.StatusInternalServerError)
			return
		}

		closeNotify := closeNotifier.CloseNotify()

		defer func() {
			delete(SSEClients, clientId)
			close(clientChannel)
			fmt.Println("Client disconnected:", clientId)
		}()

		// Continuously write messages to the client
		for {
			select {
			case message, ok := <-clientChannel:
				if !ok {
					fmt.Println("Client channel closed:", clientId)
					return
				}
				fmt.Println("Relaying info via SSE")
				fmt.Fprintf(c.Writer, "%s\n", message)
				flusher.Flush()
			case <-closeNotify:
				return
			}
		}
	}
}

// Eigene Zusatzaufgabe: Schreibe eine Funktion fourRoot welche die 4. Wurzel einer float64 berechnet und als float64 zurückgibt
func fourRoot(x float64) float64 {
	var result = 1.0
	return result
}

func main() {

	// Verzeichnis "./requests" anlegen, falls es nicht existiert
	if err := createRequestsDirectory(); err != nil {
		log.Fatal("Fehler beim Anlegen des Verzeichnisses:", err)
	}

	restoreRequests() // Anfragen einmal beim Programmstart laden

	// Default Instanz der Gin-Engine erstellen
	router := gin.Default()

	//http.HandleFunc("/sse", sseHandler)
	//http.ListenAndServe(":8080", nil)

	var messageChan = make(chan Request)
	defer close(messageChan)

	// Admin interface endpoint
	//requests2 := make(chan Request)
	//router.GET("/admin", AdminHandler(requests2))

	// Der Server soll auf der URL /requests auf alle Anfragen mit der Methode: requestCounter reagieren
	router.Any("/requests", requestCounter)

	// Serve static files from the "static-files" directory
	router.StaticFS("/static", http.Dir("./static-files"))

	// Der Server soll auf allen URL-Endpunkten mit der Methode handleTestRequest reagieren
	router.Use(handleTestRequest(messageChan))

	// Zweite Default Instanz der Gin-Engine erstellen: Management-API
	managementRouter := gin.Default()
	// Der Server soll auf der URL /view-requests auf Get Anfragen mit der Methode: viewRequests reagieren

	managementRouter.GET("/view-requests", viewRequests)
	// Füge die SSE-Route hinzu
	managementRouter.GET("/sse", SSEHandler(messageChan))

	// Hier wird der HTTP-Server mit dem Router managementRouter gestartet und auf dem Port 8081 gehostet.
	go func() {
		if err := managementRouter.Run(":8081"); err != nil {
			log.Fatal(err)
		}
	}()

	// Hier wird der HTTP-Server mit dem Router managementRouter gestartet und auf dem Port 8080 gehostet.
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}

}
