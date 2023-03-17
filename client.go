package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

var connectionObject net.PacketConn
var serverAddress *net.UDPAddr

// Функция всплывашки (должна быть запущена ОТДЕЛЬНЫМ ПРОЦЕССОМ)
func alarmio(text string) {
	window, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	lbl, _ := gtk.LabelNew(text)
	window.Add(lbl)
	window.SetSizeRequest(100, 100)
	window.ShowAll()
	time.Sleep(time.Second * 10)
	lbl.Destroy()
	window.Destroy()
}

// Функция прорисовки ареи
func drawCallback(da *gtk.DrawingArea, cr *cairo.Context, isActive bool, posDotX float64, posDotY float64) {
	if isActive {
		cr.SetSourceRGB(255, 0, 0)
		cr.Rectangle(posDotX, posDotY, 100, 100)
		cr.Fill()
	} else {
		surf, _ := cairo.NewSurfaceFromPNG("psina.png")
		cr.SetSourceSurface(surf, posDotX, posDotY)
		cr.Paint()
	}
}

// Главное окно
func main_window(conn_ptr *net.Conn) {
	conn := *conn_ptr
	var positionX float64
	var positionY float64
	var isActive bool
	// isActive, wasShot
	statePacket := make([]byte, 2)

	window, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	outerBox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 40)
	drawArea, err := gtk.DrawingAreaNew()
	if err != nil {
		log.Fatal(err)
	}
	drawArea.SetSizeRequest(400, 400)
	drawArea.Connect("draw", func(da *gtk.DrawingArea, cr *cairo.Context) {
		drawCallback(da, cr, isActive, positionX, positionY)
	})
	outerBox.Add(drawArea)

	// Добавляем обработку события BUTTON_PRESS,
	// тк по умолчанию оно не обрабатывается
	drawArea.AddEvents(int(gdk.BUTTON_PRESS_MASK))
	drawArea.Connect("button-press-event", func(_ any, event *gdk.Event) {
		btnEvent := gdk.EventButtonNewFromEvent(event)
		fmt.Printf("Clicked on %v %v (XY)\n", btnEvent.X(), btnEvent.Y())

		// Проверяем, попал ли курсор внутрь
		xCorrect := positionX < btnEvent.X() && positionX+100 > btnEvent.X()
		yCorrect := positionY < btnEvent.Y() && positionY+100 > btnEvent.Y()

		if xCorrect && yCorrect {
			fmt.Println("Bullseye")
			// Работает только если достали ствол
			if isActive {
				statePacket[1] = 1
				_, err := conn.Write(statePacket)
				if err != nil {
					log.Print(err)
				}
			}
		}
	})

	actionButton, err := gtk.ButtonNewWithLabel("Достать ствол")
	if err != nil {
		log.Fatal(err)
	}
	// Update position and color
	randomizeSource := rand.NewSource(time.Now().UnixNano())
	randomGetter := rand.New(randomizeSource)
	positionX = float64(randomGetter.Intn(400 - 80))
	positionY = float64(randomGetter.Intn(400 - 80))

	actionButton.Connect("clicked", func() {
		if !isActive {
			statePacket[0] = 1
			statePacket[1] = 0
			_, err := conn.Write(statePacket)
			if err != nil {
				log.Print(err)
			}
		}
	})

	outerBox.Add(actionButton)
	window.Add(outerBox)
	window.SetDefaultSize(700, 700)
	window.Connect("destroy", gtk.MainQuit)
	window.ShowAll()
	log.Println("Showed window")
	window.QueueDraw()
	log.Println("Draw queued")
	incomingPacket := make([]byte, 2)
	// Запускаем обработку постапующих пакетов
	go func() {
		for {
			log.Println("Waiting for incoming packet (routine)")
			_, err := conn.Read(incomingPacket)
			if err != nil {
				log.Print(err)
				return
			}
			switch incomingPacket[0] {
			case 0:
				isActive = false
			case 1:
				isActive = true
				if incomingPacket[1] == 1 {
					fmt.Println("VAS ZAVALILI")
					go alarmio("VAS ZAVALILI")
				}
			}
		}
	}()
}

// usage: ./main.exe [address]
func main() {
	gtk.Init(nil)
	if len(os.Args) < 2 {
		fmt.Println("Args: [server addr] [username]")
		os.Exit(0)
	}

	menu_window, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	grid, _ := gtk.GridNew()
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)
	lbl, _ := gtk.LabelNew(fmt.Sprintf("Выберите соперника на сервере %s", os.Args[1]))
	grid.Add(lbl)

	// Once we connect, send our name
	conn, _ := net.Dial("tcp", os.Args[1])
	conn.Write([]byte(os.Args[2]))
	time.Sleep(time.Second)
	// Отправляем сигнал о намерении получить инфу о противниках
	// conn.Write([]byte{2, 0})

	nameBuffer := make([]byte, 60000)

	// getting name
	n, err := conn.Read(nameBuffer)
	var namesList []string
	err = json.Unmarshal(nameBuffer[:n], &namesList)
	if err != nil {
		log.Println(err)
	}
	for _, el := range namesList {
		playerBtn, _ := gtk.ButtonNewWithLabel(el)
		playerBtn.Connect("clicked", func(b *gtk.Button) {
			// на нажатие отправляем имя выбранного игрока
			conn.Write([]byte(el))
			menu_window.Close()
			main_window(&conn)
		})
		grid.Add(playerBtn)
		fmt.Println("Player added")
	}

	menu_window.Connect("destroy", func() {
		fmt.Println("eexit")
		// gtk.MainQuit()
	})
	menu_window.Add(grid)
	menu_window.SetDefaultSize(400, 400)
	menu_window.ShowAll()
	gtk.Main()
}
