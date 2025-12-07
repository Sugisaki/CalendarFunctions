package holidayapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/appcheck"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var appCheckClient *appcheck.Client

func init() {
	ctx := context.Background()
	// 異なるプロジェクトのApp Checkトークンを検証するため、そのプロジェクトIDを指定します
	conf := &firebase.Config{ProjectID: "tianmingcalendar"} // 天命カレンダー
	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
		log.Fatalf("[Error] error initializing app: %v\n", err)
	}

	appCheckClient, err = app.AppCheck(ctx)
	if err != nil {
		log.Fatalf("[Error] error initializing app check: %v\n", err)
	}

	functions.HTTP("HandleHolidayRequest", HandleHolidayRequest)
}

type Holiday struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

type Response struct {
	Month    string    `json:"month"`
	Holidays []Holiday `json:"holidays"`
}

// HandleHolidayRequest は ?date=YYYY-MM-DD を受け取り、その月の祝日を返す
func HandleHolidayRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ---- App Check Verification ----
	appCheckToken := r.Header.Get("X-Firebase-AppCheck")
	if appCheckToken == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	_, err := appCheckClient.VerifyToken(appCheckToken)
	if err != nil {
		log.Printf("[Error] App Check token verification failed: %v", err)
		log.Printf("[Debug] App Check token ===> %v", appCheckToken)
		log.Printf("[Debug] Date ===> %v", r.URL.Query().Get("date"))

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// ---- 入力日付を取得 ----
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		http.Error(w, "date parameter required: e.g. ?date=2025-03-10", http.StatusBadRequest)
		return
	}

	inputDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "invalid date format (use YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	// ---- 対象月の開始・終了 ----
	firstDay := time.Date(inputDate.Year(), inputDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	// ---- Google Calendar API クライアント作成（サービスアカウント自動認証） ----
	srv, err := calendar.NewService(ctx, option.WithScopes(calendar.CalendarReadonlyScope))
	if err != nil {
		http.Error(w, "calendar service error", http.StatusInternalServerError)
		log.Println("[Error] calendar service error:", err)
		return
	}

	// ---- 日本の祝日カレンダー ID ----
	holidayCalendarID := "ja.japanese.official#holiday@group.v.calendar.google.com"

	// ---- 祝日を取得 ----
	events, err := srv.Events.List(holidayCalendarID).
		TimeMin(firstDay.Format(time.RFC3339)).
		TimeMax(lastDay.Format(time.RFC3339)).
		Do()
	if err != nil {
		http.Error(w, "calendar API error", http.StatusInternalServerError)
		log.Println("[Error] calendar API error:", err)
		return
	}

	holidays := []Holiday{}

	for _, e := range events.Items {
		holidays = append(holidays, Holiday{
			Name: e.Summary,
			Date: e.Start.Date,
		})
	}

	// ---- 出力 ----
	resp := Response{
		Month:    firstDay.Format("2006-01"),
		Holidays: holidays,
	}

	jsonData, _ := json.MarshalIndent(resp, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

