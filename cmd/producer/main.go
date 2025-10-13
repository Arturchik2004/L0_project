package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"L0_project/internal/model"
)

func main() {
	topic := "orders"
	brokerAddress := "localhost:9092"

	w := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer func() {
		if err := w.Close(); err != nil {
			log.Printf("ошибка при закрытии писателя Kafka: %v", err)
		}
	}()

	byteValue, err := os.ReadFile("./model.json")
	if err != nil {
		log.Printf("внимание: не удалось прочитать model.json (будет использоваться fake режим): %v", err)
	}

	var baseOrder model.Order
	if len(byteValue) > 0 {
		if err := json.Unmarshal(byteValue, &baseOrder); err != nil {
			log.Printf("внимание: не удалось разобрать model.json: %v", err)
		}
	}

	fmt.Println("Продюсер запущен. Нажмите CTRL+C для остановки.")
	gofakeit.Seed(time.Now().UnixNano())

	mode := os.Getenv("PRODUCER_MODE") // "json" или "fake" (default)
	for {
		var order model.Order
		if mode == "json" && baseOrder.OrderUID != "" {
			// используем шаблон из model.json, но обновим динамические поля
			order = baseOrder
			order.OrderUID = uuid.New().String()
			order.TrackNumber = fmt.Sprintf("WBILM%d", gofakeit.Number(1000000, 9999999))
			order.DateCreated = time.Now()
			// обновляем payment.transaction если есть
			order.Payment.Transaction = order.OrderUID
			// обновляем track_number в items
			for i := range order.Items {
				order.Items[i].TrackNumber = order.TrackNumber
			}
			// Нормализуем телефон в E.164, если он не в корректном формате
			order.Delivery.Phone = ensureE164(order.Delivery.Phone)
		} else {
			order = generateRandomOrder()
		}
		orderBytes, err := json.Marshal(order)
		if err != nil {
			log.Printf("не удалось преобразовать заказ в JSON: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		orderUID := order.OrderUID
		if orderUID == "" {
			orderUID = uuid.New().String()
		}

		if err := w.WriteMessages(context.Background(), kafka.Message{Key: []byte(orderUID), Value: orderBytes}); err != nil {
			log.Printf("ошибка отправки сообщения в Kafka: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		fmt.Printf("Отправлен заказ с UID: %s\n", orderUID)
		time.Sleep(2 * time.Second)
	}
}

func generateRandomOrder() model.Order {
	orderUID := uuid.New().String()
	trackNumber := fmt.Sprintf("WBILM%d", gofakeit.Number(1000000, 9999999))

	return model.Order{
		OrderUID:    orderUID,
		TrackNumber: trackNumber,
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name:    gofakeit.Name(),
			Phone:   generateE164Phone(),
			Zip:     gofakeit.Zip(),
			City:    gofakeit.City(),
			Address: gofakeit.Address().Address,
			Region:  gofakeit.State(),
			Email:   gofakeit.Email(),
		},
		Payment: model.Payment{
			Transaction:  orderUID,
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       gofakeit.Number(100, 5000),
			PaymentDt:    time.Now().Unix(),
			Bank:         "alpha",
			DeliveryCost: gofakeit.Number(100, 1500),
			GoodsTotal:   gofakeit.Number(1, 10),
			CustomFee:    0,
		},
		Items: []model.Item{
			{
				ChrtID:      gofakeit.Number(100000, 999999),
				TrackNumber: trackNumber,
				Price:       gofakeit.Number(50, 2000),
				Name:        gofakeit.ProductName(),
				Sale:        gofakeit.Number(0, 50),
				Size:        "0",
				TotalPrice:  gofakeit.Number(50, 2000),
				NmID:        gofakeit.Number(1000000, 9999999),
				Brand:       gofakeit.Company(),
				Status:      202,
			},
		},
		Locale:      "en",
		CustomerID:  gofakeit.Username(),
		DateCreated: time.Now(),
	}
}

// generateE164Phone создает псевдо-реалистичный номер телефона в формате E.164.
// Простая реализация: префикс +7 и 10 цифр. Можно улучшить при необходимости.
func generateE164Phone() string {
	digits := gofakeit.Numerify("##########")
	return "+7" + digits
}

// ensureE164 выполняет простую нормализацию телефона к E.164: удаляет все нецифровые символы
// и добавляет префикс +, предполагая код страны 7 для 10-значных номеров.
func ensureE164(phone string) string {
	if phone == "" {
		return generateE164Phone()
	}
	// если уже начинается с '+', считаем, что формат корректен
	if len(phone) > 0 && phone[0] == '+' {
		return phone
	}
	re := regexp.MustCompile(`[^0-9]`)
	digits := re.ReplaceAllString(phone, "")
	if digits == "" {
		return generateE164Phone()
	}
	// если 10 цифр — добавляем код +7
	if len(digits) == 10 {
		return "+7" + digits
	}
	// если уже содержит код страны (например, 11 и начинается с 7/8) — корректируем
	if len(digits) == 11 && (digits[0] == '7' || digits[0] == '8') {
		return "+" + digits[0:11]
	}
	// иначе просто возвращаем '+' + digits
	return "+" + digits
}
