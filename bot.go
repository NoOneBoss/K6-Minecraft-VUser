package minecraft

import (
	"log"
	"sync"
	"time"

	"github.com/Tnze/go-mc/bot"
	"github.com/Tnze/go-mc/bot/basic"
	"github.com/Tnze/go-mc/bot/msg"
	"github.com/Tnze/go-mc/bot/playerlist"
	"github.com/Tnze/go-mc/chat"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/minecraft", &MinecraftModule{})
}

// MinecraftModule представляет корневой модуль для k6
type MinecraftModule struct{}

// NewBot создаёт новый экземпляр бота для каждого VU
func (m *MinecraftModule) NewBot() *MinecraftBot {
	return &MinecraftBot{}
}

// MinecraftBot управляет состоянием и взаимодействием с Minecraft сервером
type MinecraftBot struct {
	mu              sync.Mutex
	client          *bot.Client
	player          *basic.Player
	chatHandler     *msg.Manager
	lastMessage     string
	lastHealth      float32
	isConnected     bool
	healthUpdated   chan struct{} // Канал для оповещения об обновлении здоровья
	messageReceived chan struct{} // Канал для оповещения о новом сообщении
}

// Connect подключается к Minecraft серверу
func (b *MinecraftBot) Connect(address, name, uuid, token string) error {
	b.healthUpdated = make(chan struct{}, 1)
	b.messageReceived = make(chan struct{}, 1)

	b.client = bot.NewClient()
	b.client.Auth = bot.Auth{
		Name: name,
		UUID: uuid,
		AsTk: token,
	}

	// Инициализация обработчиков событий
	b.player = basic.NewPlayer(b.client, basic.DefaultSettings, basic.EventsListener{
		GameStart:    func() error { return b.onGameStart() },
		Disconnect:   func(reason chat.Message) error { return b.onDisconnect(reason) },
		HealthChange: func(h float32, f int32, s float32) error { return b.onHealthChange(h, f, s) },
		Death:        func() error { return b.onDeath() },
	})

	b.chatHandler = msg.New(b.client, b.player, playerlist.New(b.client), msg.EventsHandler{
		PlayerChatMessage: func(msg chat.Message, validated bool) error { return b.onPlayerMessage(msg, validated) },
	})

	err := b.client.JoinServer(address)
	if err != nil {
		return err
	}

	b.isConnected = true
	go b.handleGamePackets()
	return nil
}

// SendMessage отправляет сообщение в чат
func (b *MinecraftBot) SendMessage(text string) error {
	return b.chatHandler.SendMessage(text)
}

// GetLastMessage возвращает последнее полученное сообщение
func (b *MinecraftBot) GetLastMessage() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastMessage
}

// GetHealth возвращает текущее здоровье бота
func (b *MinecraftBot) GetHealth() float32 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastHealth
}

// WaitForHealth ожидает обновления здоровья с таймаутом (в миллисекундах)
func (b *MinecraftBot) WaitForHealth(timeout int) bool {
	select {
	case <-b.healthUpdated:
		return true
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return false
	}
}

// WaitForMessage ожидает нового сообщения с таймаутом (в миллисекундах)
func (b *MinecraftBot) WaitForMessage(timeout int) bool {
	select {
	case <-b.messageReceived:
		return true
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return false
	}
}

// handleGamePackets обрабатывает входящие пакеты игры
func (b *MinecraftBot) handleGamePackets() {
	for b.isConnected {
		if err := b.client.HandleGame(); err != nil {
			log.Printf("Ошибка обработки пакетов: %v", err)
			b.mu.Lock()
			b.isConnected = false
			b.mu.Unlock()
			return
		}
	}
}

// Обработчики событий
func (b *MinecraftBot) onGameStart() error {
	log.Println("Бот присоединился к игре")
	return b.SendMessage("Привет от k6 бота!")
}

func (b *MinecraftBot) onPlayerMessage(msg chat.Message, validated bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastMessage = msg.String()
	select {
	case b.messageReceived <- struct{}{}:
	default:
	}
	return nil
}

func (b *MinecraftBot) onHealthChange(health float32, _ int32, _ float32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastHealth = health
	select {
	case b.healthUpdated <- struct{}{}:
	default:
	}
	return nil
}

func (b *MinecraftBot) onDeath() error {
	log.Println("Бот умер")
	go func() {
		time.Sleep(5 * time.Second)
		b.player.Respawn()
	}()
	return nil
}

func (b *MinecraftBot) onDisconnect(reason chat.Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.isConnected = false
	log.Printf("Отключено: %v", reason)
	return nil
}
