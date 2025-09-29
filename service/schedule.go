package service

import (
	"bytes"
	"log"
	"trackerbot/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func SendSchedule(bot *tgbotapi.BotAPI, chatID int64, items []repository.MaxRepsHistoryItem) error {
	
	points := make(plotter.XYs, len(items))

	for i := range items {
		points[i].X = float64(i + 1)
		points[i].Y = float64(items[len(items) - 1 - i].MaxReps)
	}

	// Создаем график
	p := plot.New()
	p.Title.Text = "Количество / Дни"
	p.X.Label.Text = "Дни фиксации прогресса"
	p.Y.Label.Text = "Количество отжиманий"

	line, _ := plotter.NewLine(points)
	scatter, _ := plotter.NewScatter(points)

	p.Add(line, scatter)

	// Рендерим в bytes.Buffer
	var buf bytes.Buffer
	writerTo, err := p.WriterTo(8*vg.Inch, 4*vg.Inch, "png")
	if err != nil {
		log.Fatal(err)
	}

	_, err = writerTo.WriteTo(&buf)
	if err != nil {
		return err
	}

		// Отправляем в телеграм
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
		Name:  "progress.png",
		Bytes: buf.Bytes(),
	})
	photo.Caption = "График прогресса максимальных отжиманий."

	if _, err := bot.Send(photo); err != nil {
		log.Fatal(err)
	}
	return nil

}
