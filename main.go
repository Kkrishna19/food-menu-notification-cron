package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

func main() {
	sheetID := os.Getenv("GOOGLE_SHEET_ID")
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID") // ACxxxxxx
	apiKeySid := os.Getenv("TWILIO_API_KEY")      // SKxxxxxx
	apiKeySecret := os.Getenv("TWILIO_API_SECRET")
	from := os.Getenv("TWILIO_WHATSAPP_FROM")
	to := os.Getenv("TWILIO_WHATSAPP_TO")

	// Map weekday+session to GID
	gidMap := map[string]string{
		"Mon-morning": "0",
		"Mon-evening": "1858011384",
		"Tue-morning": "4995562",
		"Tue-evening": "1729354562",
		"Wed-morning": "1635083019",
		"Wed-evening": "1546221997",
		"Thu-morning": "431745437",
		"Thu-evening": "1533556077",
		"Fri-morning": "777189077",
		"Fri-evening": "1421776393",
		"Sat-morning": "1679092649",
		"Sat-evening": "305987307",
	}

	// Get current IST time
	loc, _ := time.LoadLocation("Asia/Kolkata")
	now1 := time.Now().In(loc)
	// now := time.Now().In(loc)
	now := now1.Add(24 * time.Hour)

	day := now.Weekday().String()[:3] // Mon, Tue, ...
	var session string
	if now.Hour() < 12 {
		session = "morning"
	} else {
		session = "evening"
	}

	sheetKey := fmt.Sprintf("%s-%s", day, session)
	gid, ok := gidMap[sheetKey]
	if !ok {
		log.Printf("No GID mapping found for %s", sheetKey)
		return
	}

	// Build CSV export URL
	url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=csv&gid=%s", sheetID, gid)

	// Fetch CSV
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("failed to fetch sheet: %v", err)
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("failed to read CSV: %v", err)
	}

	// Process rows
	for i, row := range records {
		if i == 0 {
			continue // skip header
		}
		if len(row) < 6 {
			continue
		}

		menu := row[0]

		// Sum roti counts
		totalRoti := 0
		for j := 1; j <= 4; j++ {
			val, _ := strconv.Atoi(row[j])
			totalRoti += val
		}

		// Rice
		rice := 0.0
		if len(row) > 5 {
			val, err := strconv.ParseFloat(row[5], 64)
			if err == nil {
				rice = val
			}
		}

		//HX4c637d077b46f164f4ed1e98d731e168
		twilioMsg := ""
		if rice > 0 {
			twilioMsg = fmt.Sprintf("Didi aaj %s bana dijiyega , aur %s roti/paratha aur %.2f glass rice bana dijiyega", menu, strconv.Itoa(totalRoti), rice)
		} else {
			twilioMsg = fmt.Sprintf("Didi aaj %s bana dijiyega , aur %s roti/paratha bana dijiyega", menu, strconv.Itoa(totalRoti))
		}

		client := twilio.NewRestClientWithParams(twilio.ClientParams{
			Username:   apiKeySid,
			Password:   apiKeySecret,
			AccountSid: accountSid,
		})

		// Your WhatsApp number

		params := &openapi.CreateMessageParams{}
		params.SetTo(to)
		params.SetFrom(from)
		params.SetBody(twilioMsg)

		_, err = client.Api.CreateMessage(params)
		if err != nil {
			log.Fatalf("failed to send WhatsApp: %v", err)
		}

		fmt.Println("✅ WhatsApp message sent successfully")

		fmt.Println("WhatsApp message sent successfully ✅")

		fmt.Printf("Menu: %s | Total : %d | Rice: %f\n", menu, totalRoti, rice)
	}
}
