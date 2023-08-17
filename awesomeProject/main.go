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

func handleTestRequest(c *gin.Context) {
	// Gebe "Hello World" unter dem Statuscode 200 aus
	c.String(200, "Hello World\n")
	c.String(200, "Hello Universe")

	// Rufe die saveRequest Methode mit einem in eine Struct umgewandelten Request
	saveRequest(parseRequest(c))
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

func sseHandler(c *gin.Context) {
	// Check if the client supports SSE
	//if c.Request.Header.Get("Accept") != "text/event-stream" {
	//	c.JSON(http.StatusBadRequest, gin.H{"error": "SSE not supported"})
	//	return
	//}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	closeNotifier, ok := c.Writer.(http.CloseNotifier)
	if !ok {
		http.Error(c.Writer, "Server-Sent Events are not supported.", http.StatusInternalServerError)
		return
	}

	closeNotify := closeNotifier.CloseNotify()
	for {
		select {
		case <-closeNotify:
			fmt.Println("Client disconnected")
			return
		case <-time.After(1 * time.Second):
			// Simulate sending SSE events
			eventData := "data: This is an SSE event\n\n"
			c.Writer.WriteString(eventData)
			c.Writer.Flush()
		}
	}
}

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

	c.JSON(200, currentRequestSlice)
}

// Zeigt eine Liste von Requests basierend auf dem Query-Parameter "p" an.
func viewRequests2(c *gin.Context) {
	// Lese alle Einträge in der Datei "./requests"
	entries, err := os.ReadDir("./requests")
	if err != nil {
		c.String(500, "Fehler beim Lesen der Dateien")
		return
	}

	// Erstelle ein Mapping von Dateinamen zu Modifikationszeiten
	modTimes := make(map[string]time.Time)

	// Erfasse die Modifikationszeiten für jeden Eintrag
	for _, entry := range entries {
		fmt.Println(entry)
		fileInfo, err := entry.Info()
		if err != nil {
			log.Println("Fehler beim Abrufen von FileInfo:", err)
			continue
		}
		modTimes[entry.Name()] = fileInfo.ModTime()
	}

	// Sortiere Dateinamen nach Modifikationsdatum absteigend
	var sortedFilenames []string
	for filename, modTime := range modTimes {
		sortedFilenames = append(sortedFilenames, filename)
		for i := len(sortedFilenames) - 1; i > 0; i-- {
			if modTimes[sortedFilenames[i-1]].Before(modTime) {
				sortedFilenames[i], sortedFilenames[i-1] = sortedFilenames[i-1], sortedFilenames[i]
			} else {
				break
			}
		}
	}

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
	if endIndex > len(sortedFilenames) {
		endIndex = len(sortedFilenames)
	}

	var currentRequestSlice []Request
	for _, filename := range sortedFilenames[startIndex:endIndex] {
		data, err := os.ReadFile(fmt.Sprintf("./requests/%s", filename))
		if err != nil {
			log.Println("Fehler beim Lesen der Datei:", err)
			continue
		}

		var req Request
		if err := json.Unmarshal(data, &req); err != nil {
			log.Println("Fehler beim Entmarshalling des Requests:", err)
			continue
		}
		currentRequestSlice = append(currentRequestSlice, req)
	}

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
func streamRequests(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Warte auf neue Anfragen und sende sie an den Client
	for {
		data := "Neue Anfrage empfangen!" // Hier kannst du die gewünschten Daten senden
		c.String(http.StatusOK, "data: %s\n\n", data)
		c.Writer.Flush()

		// Füge eine Pause zwischen den Ereignissen ein (z. B. 1 Sekunde)
		time.Sleep(1 * time.Second)
	}
}

func main() {

	// Verzeichnis "./requests" anlegen, falls es nicht existiert
	if err := createRequestsDirectory(); err != nil {
		log.Fatal("Fehler beim Anlegen des Verzeichnisses:", err)
	}

	restoreRequests() // Anfragen einmal beim Programmstart laden

	// Default Instanz der Gin-Engine erstellen
	router := gin.Default()

	// Füge die SSE-Route hinzu
	router.GET("/sse", sseHandler)

	// Der Server soll auf der URL /requests auf alle Anfragen mit der Methode: requestCounter reagieren
	router.Any("/requests", requestCounter)

	// Serve static files from the "static-files" directory
	router.StaticFS("/static", http.Dir("./static-files"))

	// Der Server soll auf allen URL-Endpunkten mit der Methode handleTestRequest reagieren
	router.Use(handleTestRequest)

	// Zweite Default Instanz der Gin-Engine erstellen: Management-API
	managementRouter := gin.Default()
	// Der Server soll auf der URL /view-requests auf Get Anfragen mit der Methode: viewRequests reagieren

	managementRouter.GET("/view-requests", viewRequests)

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
