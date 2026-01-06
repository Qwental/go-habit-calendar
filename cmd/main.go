package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/fogleman/gg"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var ghColors = []string{"#ebedf0", "#9be9a8", "#40c463", "#30a14e", "#216e39"}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	godotenv.Load("../config/.env")

	bot, _ := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	u := tgbotapi.NewUpdate(0)
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil || update.Message.Text != "/test_heatmap" {
			continue
		}

		year := 2026
		// Эмуляция данных
		entries := make(map[string]bool)
		// Пример: streak 3-13 янв, 14 пропуск, 15 ок
		for i := 3; i <= 13; i++ {
			entries[fmt.Sprintf("2026-01-%02d", i)] = true
		}
		entries["2026-01-15"] = true

		imgFile := drawYearCalendar(year, "НЕ ЕСТЬ СЛАДКУЮ ЕДУ 🍰", entries)
		photo := tgbotapi.NewPhoto(update.Message.Chat.ID, tgbotapi.FilePath(imgFile))
		bot.Send(photo)
		os.Remove(imgFile)
	}
}

func drawYearCalendar(year int, habitName string, entries map[string]bool) string {
	// Ограничиваем название (проверка в коде бота тоже нужна)
	if len(habitName) > 100 {
		habitName = habitName[:97] + "..."
	}

	const (
		multiplier = 2.0 // Коэффициент качества (увеличиваем всё в 2 раза)
		mPadding   = 60.0 * multiplier
		cellSize   = 20.0 * multiplier
		spacing    = 4.0  * multiplier
		
		mWidth     = 7*(cellSize+spacing) + 40*multiplier
		mHeight    = 6*(cellSize+spacing) + 60*multiplier
		
		canvasW    = 4*mWidth + 5*mPadding
		canvasH    = 3*mHeight + 4*mPadding + 150*multiplier // Больше места сверху под заголовок
	)

	fontPath := "../assets/fonts/GoogleSans.ttf"
	dc := gg.NewContext(int(canvasW), int(canvasH))
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// 1. Название привычки (крупный заголовок)
	dc.SetRGB(0, 0, 0)
	dc.LoadFontFace(fontPath, 48*multiplier) 
	dc.DrawStringAnchored(habitName, canvasW/2, 60*multiplier, 0.5, 0.5)

	// 2. Подзаголовок с годом
	dc.SetRGB(0.4, 0.4, 0.4)
	dc.LoadFontFace(fontPath, 24*multiplier)
	dc.DrawStringAnchored(fmt.Sprintf("Статистика за %d год", year), canvasW/2, 100*multiplier, 0.5, 0.5)

	// ... (Расчет streaks остается таким же) ...
    streaks := make(map[string]int)
    currentStreak := 0
    startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
    for d := 0; d < 366; d++ {
        t := startOfYear.AddDate(0, 0, d)
        if t.Year() != year { break }
        dateStr := t.Format("2006-01-02")
        if entries[dateStr] { currentStreak++ } else { currentStreak = 0 }
        streaks[dateStr] = currentStreak
    }

	// Рисуем месяцы
	for m := 0; m < 12; m++ {
		month := time.Month(m + 1)
		rowM := m / 4
		colM := m % 4

		baseX := mPadding + float64(colM)*(mWidth+mPadding)
		baseY := 300.0 + float64(rowM)*(mHeight+mPadding) // Опустили сетку ниже под заголовок

		// Название месяца
		dc.SetRGB(0, 0, 0)
		dc.LoadFontFace(fontPath, 22*multiplier)
		dc.DrawString(month.String(), baseX, baseY-30)

		// Дни недели
		dc.LoadFontFace(fontPath, 12*multiplier)
		dc.SetRGB(0.5, 0.5, 0.5)
		weekLabels := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
		for i, l := range weekLabels {
			dc.DrawStringAnchored(l, baseX+float64(i)*(cellSize+spacing)+cellSize/2, baseY, 0.5, 0.5)
		}

		t := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		for t.Month() == month {
			firstDayOffset := (int(time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Weekday()) + 6) % 7
			dayIdx := t.Day() - 1 + firstDayOffset
			col := dayIdx % 7
			row := dayIdx / 7

			x := baseX + float64(col)*(cellSize+spacing)
			y := baseY + 25*multiplier + float64(row)*(cellSize+spacing)

			s := streaks[t.Format("2006-01-02")]
			level := 0
			if s > 0 {
				level = int(math.Min(float64(s/3)+1, 4))
			}
			
			dc.SetHexColor(ghColors[level])
			dc.DrawRoundedRectangle(x, y, cellSize, cellSize, 4*multiplier)
			dc.Fill()

            // Номера недель
			if col == 0 {
				_, weekNum := t.ISOWeek()
				dc.SetRGB(0.7, 0.7, 0.7)
				dc.LoadFontFace(fontPath, 10*multiplier)
				dc.DrawStringAnchored(fmt.Sprintf("%d", weekNum), baseX-30, y+cellSize/2, 0.5, 0.5)
			}
			t = t.AddDate(0, 0, 1)
		}
	}

	fname := fmt.Sprintf("stats_%d.png", time.Now().Unix())
	dc.SavePNG(fname)
	return fname
}

