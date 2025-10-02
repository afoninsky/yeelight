package yeelight

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

type Yeelight struct {
	YLID            int32         `json:"id"`
	Address         string        `json:"address"`
	Persistent      bool          `json:"persistent",default0:"false"`
	Conn            net.Conn      `json:"-"`
	ConnectTimeout  time.Duration
	ResponseTimeout time.Duration
}

type Command struct {
	ID     int32       `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type Response struct {
	ID     int32       `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

type Options struct {
	Smooth int `default0:"200"`
}

type FxMode struct {
	Mode string `json:"mode"`
}

type Color struct {
	Value int64
}

type ColorMatrix struct {
	Colors []Color
}

type Vector struct {
	Row    int
	Column int
}

// FlowState defines a single state in a color flow.
type FlowState struct {
	Duration   int
	Mode       FlowMode
	Value      int
	Brightness int
}

// FlowMode defines the type of a flow state.
type FlowMode int

const (
	FlowModeColor FlowMode = 1
	FlowModeTemp  FlowMode = 2
	FlowModeSleep FlowMode = 7
)

// CfAction defines the action to take after a color flow finishes.
type CfAction int

const (
	CfActionRecover CfAction = 0 // Revert to the state before the flow
	CfActionStay    CfAction = 1 // Stay at the last state of the flow
	CfActionOff     CfAction = 2 // Turn off the light
)

func (v *Vector) Index() int {
	r := v.Column
	r += (v.Row * 5)
	return r
}

func MakeMatrix(hex string, size int) ColorMatrix {
	colorMatrix := ColorMatrix{}
	for i := 0; i < size; i++ {
		colorMatrix.Colors = append(colorMatrix.Colors, MakeColorHEX(hex))
	}

	return colorMatrix
}

func MakeSpotMatrix(hex string) ColorMatrix {
	colorMatrix := MakeMatrix("#000000", 5)
	colorMatrix.SetHex(Vector{0, 0}, hex)
	return colorMatrix
}

func MakeFromHexColors(matrix []string) ColorMatrix {
	colorMatrix := ColorMatrix{}
	for _, element := range matrix {
		colorMatrix.Colors = append(colorMatrix.Colors, MakeColorHEX(element))
	}

	return colorMatrix
}

func (matrix *ColorMatrix) ToASCII() string {
	ascii := ""
	for _, element := range matrix.Colors {
		ascii += element.ToASCII()
	}

	return ascii
}

func (matrix *ColorMatrix) ReplaceAllHex(h string) {
	for index := range matrix.Colors {
		matrix.Colors[index].Hex(h)
	}
}

func (matrix *ColorMatrix) ReplaceAllRGB(r int8, g int8, b int8) {
	for index, element := range matrix.Colors {
		element.RGB(r, g, b)
		matrix.Colors[index] = element
	}
}

func (matrix *ColorMatrix) SetHex(v Vector, h string) {
	matrix.Colors[v.Index()].Hex(h)
}

func (matrix *ColorMatrix) SetColor(v Vector, c Color) {
	matrix.Colors[v.Index()] = c
}

func (matrix *ColorMatrix) GetColor(v Vector) Color {
	return matrix.Colors[v.Index()]
}

func (matrix *ColorMatrix) SetRGB(v Vector, r int8, g int8, b int8) {
	matrix.Colors[v.Index()].RGB(r, g, b)
}

func (matrix *ColorMatrix) Rotate(angle float64) ColorMatrix {
	return matrix.RotateAt(angle, Vector{2, 2})
}

func (matrix *ColorMatrix) RotateAt(angle float64, center Vector) ColorMatrix {
	new_matrix := MakeMatrix("#000000", 25)
	a := float64(angle * math.Pi / 180.0)

	cx := float64(center.Column)
	cy := float64(center.Row)

	for y := 0.0; y < 5.0; y++ {
		for x := 0.0; x < 5.0; x++ {
			x_f_new := cx + ((x-cx)*math.Cos(a) - (y-cy)*math.Sin(a))
			y_f_new := cy + ((x-cx)*math.Sin(a) + (y-cy)*math.Cos(a))

			x_new := int(math.Round(math.Abs(x_f_new)))
			y_new := int(math.Round(math.Abs(y_f_new)))

			v_old := Vector{Column: int(x), Row: int(y)}
			v_new := Vector{Column: x_new, Row: y_new}

			color := matrix.GetColor(v_old)
			new_matrix.SetColor(v_new, color)
		}
	}

	return new_matrix
}

func MakeColorRGB(r int8, g int8, b int8) Color {
	color := Color{}
	color.RGB(r, g, b)
	return color
}

func MakeColorHEX(hex string) Color {
	color := Color{}
	color.Hex(hex)
	return color
}

func (color *Color) ToHex() string {
	return fmt.Sprintf("%06x", color.Value)
}

func (color *Color) Hex(hexv string) (err error) {
	color.Value, err = strconv.ParseInt(strings.Trim(hexv, " # "), 16, 64)

	if err != nil {
		return err
	}

	return nil
}

func (color *Color) RGB(r int8, g int8, b int8) {
	color.Value = int64(b)
	color.Value += (int64(r) << 16)
	color.Value += (int64(g) << 8)
}

func (color *Color) ToRGB() (r byte, g byte, b byte) {
	r = byte((color.Value & 0xFF0000) >> 16)
	g = byte((color.Value & 0x00FF00) >> 8)
	b = byte(color.Value & 0x0000FF)
	return
}

func (color *Color) ToASCII() (result string) {
	ASCII_TABLE := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	total_bytes := color.Value / 64
	colorValue := color.Value % 64

	var encoded_data []byte

	encoded_data = append(encoded_data, ASCII_TABLE[total_bytes/4096])
	total_bytes = total_bytes % 4096

	encoded_data = append(encoded_data, ASCII_TABLE[total_bytes/64])
	total_bytes = total_bytes % 64

	encoded_data = append(encoded_data, ASCII_TABLE[total_bytes])
	encoded_data = append(encoded_data, ASCII_TABLE[colorValue])
	result = string(encoded_data)
	return
}

func (c *Command) GenerateID() {
	if c.ID == 0 {
		r := rand.NewSource(time.Now().UnixNano())
		c.ID = rand.New(r).Int31()
	}
}

func (c *Command) ToJson() ([]byte, error) {
	cmdJson, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return cmdJson, nil
}

func (r *Response) FromJson(data []byte) error {
	return json.Unmarshal(data, &r)
}

func (yl *Yeelight) Connect() (err error) {
	if yl.ConnectTimeout == 0 {
		yl.ConnectTimeout = 3 * time.Second
	}

	yl.Conn, err = net.DialTimeout("tcp", yl.Address, yl.ConnectTimeout)
	if err != nil {
		return err
	}

	return nil
}

func (yl *Yeelight) SendCommand(c Command) (r Response, err error) {
	c.GenerateID()
	if err = yl.Connect(); err != nil {
		return
	}

	if !yl.Persistent {
		defer yl.Conn.Close()
	}

	cmdJSON, err := c.ToJson()
	if err != nil {
		return r, err
	}

	if _, err := fmt.Fprintf(yl.Conn, "%s\r\n", cmdJSON); err != nil {
		return r, err
	}

	s := make(chan string)
	e := make(chan error)

	go func() {
		reader := bufio.NewReader(yl.Conn)
		response, err := reader.ReadString('\n')
		if err != nil {
			e <- err
		} else {
			s <- response
		}
		close(s)
		close(e)
	}()

	if yl.ResponseTimeout == 0 {
		yl.ResponseTimeout = 500 * time.Millisecond
	}

	select {
	case response := <-s:
		r.FromJson([]byte(response))
		return r, nil
	case err := <-e:
		return r, err
	case <-time.After(yl.ResponseTimeout):
		return r, nil
	}
}

func (yl *Yeelight) GetProperties(names []string) (r Response, err error) {
	c := Command{
		Method: "get_prop",
		Params: names,
	}

	return yl.SendCommand(c)
}

func (yl *Yeelight) GetProperty(name string) (r Response, err error) {
	c := Command{
		Method: "get_prop",
		Params: []interface{}{name},
	}

	return yl.SendCommand(c)
}

// Wrapper Methods

func (yl *Yeelight) SetHexColor(color string, options Options) (err error) {
	color = strings.Replace(color, "#", "", -1)
	n, err := strconv.ParseUint(color, 16, 64)
	if err != nil {
		return
	}

	c := Command{
		Method: "set_rgb",
		Params: []interface{}{n, "smooth", options.Smooth},
	}

	_, err = yl.SendCommand(c)
	if err != nil {
		return
	}

	return nil
}

func (yl *Yeelight) GetHexColor() (h string, err error) {
	r, err := yl.GetProperty("rgb")
	if err != nil {
		return h, err
	}

	value, err := strconv.Atoi(r.Result.([]interface{})[0].(string))

	rgb := Color{Value: int64(value)}
	if err != nil {
		return h, err
	}
	h = rgb.ToHex()
	return h, nil
}

func (yl *Yeelight) SetBright(value int8, options Options) (err error) {
	c := Command{
		Method: "set_bright",
		Params: []interface{}{value, "smooth", options.Smooth},
	}

	_, err = yl.SendCommand(c)
	if err != nil {
		return
	}

	return nil
}

func (yl *Yeelight) SetColorTemperature(value int16, options Options) (err error) {
	c := Command{
		Method: "set_ct_abx",
		Params: []interface{}{value, "smooth", options.Smooth},
	}

	_, err = yl.SendCommand(c)
	if err != nil {
		return
	}

	return nil
}

func (yl *Yeelight) GetBright() (value int8, err error) {
	r, err := yl.GetProperty("bright")
	if err != nil {
		return value, err
	}
	v, err := strconv.ParseInt(r.Result.([]interface{})[0].(string), 10, 8)
	if err != nil {
		return value, err
	}

	value = int8(v)

	return value, nil
}

func (yl *Yeelight) SetOn(options Options) (err error) {
	c := Command{
		Method: "set_power",
		Params: []interface{}{"on", "smooth", options.Smooth},
	}

	_, err = yl.SendCommand(c)
	if err != nil {
		return
	}

	return nil
}

func (yl *Yeelight) SetOff(options Options) (err error) {
	c := Command{
		Method: "set_power",
		Params: []interface{}{"off", "smooth", options.Smooth},
	}

	_, err = yl.SendCommand(c)
	if err != nil {
		return
	}

	return nil
}

func (yl *Yeelight) Toggle() (err error) {
	c := Command{
		Method: "toggle",
		Params: []interface{}{},
	}

	_, err = yl.SendCommand(c)
	if err != nil {
		return
	}

	return nil
}

func (yl *Yeelight) IsOn() (b bool, err error) {
	r, err := yl.GetProperty("power")
	if err != nil {
		return b, err
	}

	b = (r.Result.([]interface{})[0].(string) == "on")

	return b, err
}

func (yl *Yeelight) Sleep(s int8) (err error) {
	c := Command{
		Method: "cron_add",
		Params: []interface{}{0, s},
	}

	_, err = yl.SendCommand(c)
	if err != nil {
		return
	}

	return nil
}

func (yl *Yeelight) SetMatrix(matrix []ColorMatrix) (err error) {
	ascii := ""

	for _, element := range matrix {
		ascii += element.ToASCII()
	}

	err = yl.SetASCII(ascii)

	if err != nil {
		return
	}

	return nil
}

func (yl *Yeelight) SetASCII(ascii string) (err error) {

	c := Command{
		Method: "update_leds",
		Params: []interface{}{ascii},
	}

	_, err = yl.SendCommand(c)
	if err != nil {
		return
	}

	return nil
}

func (yl *Yeelight) SetDirectMode() (err error) {
	mode := FxMode{Mode: "direct"}
	c := Command{
		Method: "activate_fx_mode",
		Params: []interface{}{mode},
	}

	_, err = yl.SendCommand(c)
	if err != nil {
		return
	}

	return nil
}

// StartCf starts a color flow.
// count: how many times to repeat the flow. 0 means infinite.
// action: what to do after the flow finishes.
// flow: a slice of FlowState structs defining the flow.
func (yl *Yeelight) StartCf(count int, action CfAction, flow []FlowState) error {
	var stateStrings []string
	for _, state := range flow {
		s := fmt.Sprintf("%d,%d,%d,%d", state.Duration, state.Mode, state.Value, state.Brightness)
		stateStrings = append(stateStrings, s)
	}
	flowExpression := strings.Join(stateStrings, ",")

	c := Command{
		Method: "start_cf",
		Params: []interface{}{count, int(action), flowExpression},
	}

	_, err := yl.SendCommand(c)
	if err != nil {
		return err
	}

	return nil
}

// StopCf stops the currently running color flow.
func (yl *Yeelight) StopCf() error {
	c := Command{
		Method: "stop_cf",
		Params: []interface{}{},
	}

	_, err := yl.SendCommand(c)
	if err != nil {
		return err
	}

	return nil
}

func (yl *Yeelight) Disconnect() {
	yl.Conn.Close()
}

// SetName sets a new name for the Yeelight.
func (yl *Yeelight) SetName(name string) error {
	c := Command{
		Method: "set_name",
		Params: []interface{}{name},
	}

	_, err := yl.SendCommand(c)
	if err != nil {
		return err
	}

	return nil
}