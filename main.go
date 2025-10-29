package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"log"
	"strconv"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/tarm/serial"
)

type UIState struct {
	LastPacket Packet
	Series     []float32
	LogLines   []string

	AvailablePorts []string
	PortList       widget.Enum
	BaudList       widget.Enum
	OpenBtn        widget.Clickable
	ClearBtn       widget.Clickable

	PortOpen bool
}

const (
	seriesCapacity = 300
	logCapacity    = 200
)

func main() {
	go runApp()
	app.Main()
}

func runApp() {
	w := new(app.Window)
	w.Option(
		app.Title("Kompiuterinės Komunikacijos 2 Lab. Darbas"),
		app.Size(unit.Dp(1100), unit.Dp(700)),
	)

	th := material.NewTheme()
	var state UIState

	state.AvailablePorts = []string{
		"COM40 - STMicroelect",
		"COM12 - USB-SERIAL",
		"/dev/tty.usbserial-0001",
	}
	state.PortList.Value = state.AvailablePorts[0]

	baudRates := []string{"115200", "921600", "460800", "9600"}
	state.BaudList.Value = baudRates[0]

	packets := make(chan Packet, 128)

	go startSerialReader(
		w,
		packets,
		&state,
	)

	for {
		e := w.Event()
		switch ev := e.(type) {

		case app.DestroyEvent:
			if ev.Err != nil {
				log.Println("window destroy:", ev.Err)
			}
			return

		case app.FrameEvent:

			var ops op.Ops
			gtx := app.NewContext(&ops, ev)

		drain:
			for {
				select {
				case p := <-packets:
					state.LastPacket = p

					v := float32(p.Acceleration[2])
					state.Series = append(state.Series, v)
					if len(state.Series) > seriesCapacity {
						state.Series = state.Series[len(state.Series)-seriesCapacity:]
					}

					line := fmt.Sprintf("%s Lat:%.6f Lon:%.6f Sat:%d AccZ:%.2f",
						p.Time, p.Latitude, p.Longitude, p.Satellites, p.Acceleration[2])
					state.LogLines = append(state.LogLines, line)
					if len(state.LogLines) > logCapacity {
						state.LogLines = state.LogLines[len(state.LogLines)-logCapacity:]
					}

				default:
					break drain
				}
			}

			if state.OpenBtn.Clicked(gtx) {

				state.PortOpen = true
				state.LogLines = append(state.LogLines,
					"[INFO] COM PORT opened: "+state.PortList.Value+
						" @ "+state.BaudList.Value+" baud")
				if len(state.LogLines) > logCapacity {
					state.LogLines = state.LogLines[len(state.LogLines)-logCapacity:]
				}
			}
			if state.ClearBtn.Clicked(gtx) {
				state.LogLines = nil
				state.Series = nil
			}

			layoutRoot(gtx, th, &state, baudRates)

			ev.Frame(gtx.Ops)
		}
	}
}

func startSerialReader(w *app.Window, out chan Packet, state *UIState) {

	baud, _ := strconv.Atoi(state.BaudList.Value)

	cfg := &serial.Config{
		Name:        "/dev/tty.usbserial-0001",
		Baud:        baud,
		Size:        8,
		Parity:      serial.ParityOdd,
		StopBits:    serial.Stop1,
		ReadTimeout: time.Millisecond * 500,
	}

	port, err := serial.OpenPort(cfg)
	if err != nil {
		log.Println("cannot open port:", err)
		return
	}
	defer port.Close()

	reader := bufio.NewReader(port)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println("read error:", err)
			continue
		}

		p, err := ParsePacket(line)
		if err != nil {
			log.Println("parse error:", err)
			continue
		}

		select {
		case out <- p:
		default:
			select {
			case <-out:
			default:
			}
			out <- p
		}

		w.Invalidate()
	}
}

func layoutRoot(gtx layout.Context, th *material.Theme, st *UIState, baudRates []string) layout.Dimensions {

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {

			border := widgetBorder(gtx, color.NRGBA{R: 180, G: 0, B: 0, A: 255})
			return border(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(0.4, func(gtx layout.Context) layout.Dimensions {
						return sectionPortHeader(gtx, th)
					}),
					layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
						return sectionGPSHeader(gtx, th, st)
					}),
					layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
						return sectionTimeHeader(gtx, th, st)
					}),
					layout.Flexed(0.2, func(gtx layout.Context) layout.Dimensions {
						return sectionSatsHeader(gtx, th, st)
					}),
				)
			})
		}),

		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,

				layout.Flexed(0.4, func(gtx layout.Context) layout.Dimensions {
					border := widgetBorder(gtx, color.NRGBA{R: 180, G: 0, B: 0, A: 255})
					return border(func(gtx layout.Context) layout.Dimensions {
						return leftPanel(gtx, th, st, baudRates)
					})
				}),

				layout.Flexed(0.6, func(gtx layout.Context) layout.Dimensions {
					border := widgetBorder(gtx, color.NRGBA{R: 180, G: 0, B: 0, A: 255})
					return border(func(gtx layout.Context) layout.Dimensions {
						return rightPanel(gtx, th, st)
					})
				}),
			)
		}),
	)
}

func sectionPortHeader(gtx layout.Context, th *material.Theme) layout.Dimensions {

	return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Body1(th, "COM PORT Valdymas").Layout(gtx)
	})
}

func sectionGPSHeader(gtx layout.Context, th *material.Theme, st *UIState) layout.Dimensions {
	txt := fmt.Sprintf("GPS Koordinatės:\n%.6f, %.6f",
		st.LastPacket.Latitude, st.LastPacket.Longitude)
	if st.LastPacket.Latitude == 0 && st.LastPacket.Longitude == 0 {
		txt = "GPS Koordinatės:\n-----"
	}
	return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Body2(th, txt).Layout(gtx)
	})
}

func sectionTimeHeader(gtx layout.Context, th *material.Theme, st *UIState) layout.Dimensions {
	timeTxt := st.LastPacket.Time
	if timeTxt == "" {
		timeTxt = "-----"
	}
	txt := "EET Laikas:\n" + timeTxt
	return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Body2(th, txt).Layout(gtx)
	})
}

func sectionSatsHeader(gtx layout.Context, th *material.Theme, st *UIState) layout.Dimensions {
	sats := "-----"
	if st.LastPacket.Satellites != 0 {
		sats = fmt.Sprintf("%d", st.LastPacket.Satellites)
	}
	txt := "Palydovų skaičius:\n" + sats
	return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Body2(th, txt).Layout(gtx)
	})
}

func leftPanel(gtx layout.Context, th *material.Theme, st *UIState, baudRates []string) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return comControls(gtx, th, st, baudRates)
			})
		}),

		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {

				bg := color.NRGBA{R: 230, G: 230, B: 230, A: 255}
				paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

				inset := layout.UniformInset(unit.Dp(8))
				return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {

					children := make([]layout.FlexChild, 0, len(st.LogLines))
					for i := len(st.LogLines) - 1; i >= 0; i-- {
						line := st.LogLines[i]
						ch := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Body2(th, line).Layout(gtx)
						})
						children = append(children, ch)
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
				})
			})
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(th, &st.ClearBtn, "Išvalyti duomenis")
				return btn.Layout(gtx)
			})
		}),
	)
}

func comControls(gtx layout.Context, th *material.Theme, st *UIState, baudRates []string) layout.Dimensions {

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return labeledRow(gtx, th, "PORT pasirinkimas:", st.PortList.Value)
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return labeledRow(gtx, th, "Baud Rate pasirinkimas:", st.BaudList.Value)
		}),

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &st.OpenBtn, "Atidaryti COM PORT")
			return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, btn.Layout)
		}),
	)
}

func labeledRow(gtx layout.Context, th *material.Theme, label, value string) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Body1(th, label).Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			b := material.Body1(th, value)
			return b.Layout(gtx)
		}),
	)
}

func rightPanel(gtx layout.Context, th *material.Theme, st *UIState) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,

		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			inset := layout.UniformInset(unit.Dp(8))
			return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				title := material.H6(th, "Duomenų grafikas")
				return title.Layout(gtx)
			})
		}),

		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {

			inset := layout.UniformInset(unit.Dp(16))
			return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				wPx := gtx.Constraints.Max.X
				hPx := gtx.Constraints.Max.Y

				minH := gtx.Dp(unit.Dp(200))
				if hPx < minH {
					hPx = minH
				}

				c := gtx.Constraints
				c.Min.Y = hPx
				c.Max.Y = hPx
				gtx.Constraints = c

				return drawGraph(gtx, st.Series, wPx, hPx)
			})
		}),
	)
}

func drawGraph(gtx layout.Context, series []float32, width, height int) layout.Dimensions {

	paint.FillShape(
		gtx.Ops,
		color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		clip.Rect{Max: image.Pt(width, height)}.Op(),
	)

	if len(series) < 2 {
		return layout.Dimensions{Size: image.Pt(width, height)}
	}

	minV, maxV := series[0], series[0]
	for i := 1; i < len(series); i++ {
		if series[i] < minV {
			minV = series[i]
		}
		if series[i] > maxV {
			maxV = series[i]
		}
	}
	if maxV-minV < 0.1 {
		minV = -0.05
		maxV = 0.05
	}

	leftPad := float32(40)
	rightPad := float32(20)
	topPad := float32(20)
	bottomPad := float32(40)

	plotW := float32(width) - leftPad - rightPad
	plotH := float32(height) - topPad - bottomPad
	if plotW <= 0 || plotH <= 0 {
		return layout.Dimensions{Size: image.Pt(width, height)}
	}

	{
		var box clip.Path
		box.Begin(gtx.Ops)
		box.MoveTo(f32.Pt(leftPad, topPad))
		box.LineTo(f32.Pt(leftPad+plotW, topPad))
		box.LineTo(f32.Pt(leftPad+plotW, topPad+plotH))
		box.LineTo(f32.Pt(leftPad, topPad+plotH))
		box.Close()

		paint.FillShape(
			gtx.Ops,
			color.NRGBA{R: 0, G: 0, B: 0, A: 255},
			clip.Stroke{
				Path:  box.End(),
				Width: 1,
			}.Op(),
		)
	}

	if minV < 0 && maxV > 0 {
		ynorm := (0 - minV) / (maxV - minV)
		y0 := topPad + (1-ynorm)*plotH

		var axis clip.Path
		axis.Begin(gtx.Ops)
		axis.MoveTo(f32.Pt(leftPad, y0))
		axis.LineTo(f32.Pt(leftPad+plotW, y0))

		paint.FillShape(
			gtx.Ops,
			color.NRGBA{R: 180, G: 180, B: 180, A: 255},
			clip.Stroke{
				Path:  axis.End(),
				Width: 1,
			}.Op(),
		)
	}

	var sig clip.Path
	sig.Begin(gtx.Ops)

	n := len(series)
	for i := 0; i < n; i++ {
		xn := float32(i) / float32(n-1)
		x := leftPad + xn*plotW

		vnorm := (series[i] - minV) / (maxV - minV)
		y := topPad + (1-vnorm)*plotH

		if i == 0 {
			sig.MoveTo(f32.Pt(x, y))
		} else {
			sig.LineTo(f32.Pt(x, y))
		}
	}

	paint.FillShape(
		gtx.Ops,
		color.NRGBA{R: 33, G: 150, B: 243, A: 255},
		clip.Stroke{
			Path:  sig.End(),
			Width: 2,
		}.Op(),
	)

	return layout.Dimensions{Size: image.Pt(width, height)}
}

func widgetBorder(gtx layout.Context, col color.NRGBA) func(func(layout.Context) layout.Dimensions) layout.Dimensions {
	return func(child func(layout.Context) layout.Dimensions) layout.Dimensions {

		var dims layout.Dimensions

		macro := op.Record(gtx.Ops)
		dims = child(gtx)
		call := macro.Stop()

		var rect clip.Path
		rect.Begin(gtx.Ops)
		rect.MoveTo(f32.Pt(0, 0))
		rect.LineTo(f32.Pt(float32(dims.Size.X), 0))
		rect.LineTo(f32.Pt(float32(dims.Size.X), float32(dims.Size.Y)))
		rect.LineTo(f32.Pt(0, float32(dims.Size.Y)))
		rect.Close()

		paint.FillShape(
			gtx.Ops,
			col,
			clip.Stroke{
				Path:  rect.End(),
				Width: 1,
			}.Op(),
		)

		call.Add(gtx.Ops)
		return dims
	}
}
