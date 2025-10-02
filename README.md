# MCP Yeelight Cube Controller

A scripting system for controlling Yeelight Cube smart lamp with simple text-based commands.

## Usage

```bash
go run main.go <script_name> [interval_ms] [timeout_s]
```

### Parameters:
- `script_name`: Name of the script (without .txt extension)
- `interval_ms`: Interval between frames in milliseconds (default: 500)
- `timeout_s`: Timeout in seconds (default: 0 = infinite, press Enter to stop)

### Environment Variables:
- `YEELIGHT_ADDR`: Yeelight address (default: 192.168.1.118:55443)
- `YEELIGHT_SCRIPTS`: Path to scripts folder (default: ./scripts)

### Examples:

```bash
# Run spinner animation with 200ms interval for 10 seconds
go run main.go spinner 200 10

# Run wave effect with custom Yeelight address
YEELIGHT_ADDR=192.168.1.100:55443 go run main.go wave

# Run corners animation from custom scripts path
YEELIGHT_SCRIPTS=/home/user/my-scripts go run main.go corners 300
```

## Available Scripts

- **spinner**: Animated spinner
- **green**: Static green circle
- **pulse**: Pulsing center dot
- **wave**: Horizontal wave effect
- **cross**: Animated cross pattern
- **checkerboard**: Static checkerboard
- **slide**: Sliding line animation
- **corners**: Blinking corners
- **rotate_square**: Rotating square
- **fade**: Fading effect

## Script Language

Scripts use a simple text-based language to control the 5x5 LED matrix.

### Commands:

- `FILL <color>`: Fill entire matrix with color
- `PIXEL <x> <y> <color>`: Set single pixel (0-4, 0-4)
- `RECT <x1> <y1> <x2> <y2> <color>`: Draw filled rectangle
- `LINE <direction> <position> <color>`: Draw horizontal/vertical line
  - direction: H (horizontal) or V (vertical)
  - position: 0-4
- `CIRCLE <x> <y> <radius> <color>`: Draw filled circle
- `CROSS <x> <y> <size> <color>`: Draw cross pattern
- `FRAME`: Mark end of frame (for animations)

### Colors:
- Hex format: `#FF0000` (red), `#00FF00` (green), `#0000FF` (blue)
- Black/off: `#000000`

### Example Script:

```
# Simple spinner animation
FILL #000000
PIXEL 2 0 #FF0000
FRAME

FILL #000000  
PIXEL 4 0 #FF0000
FRAME

FILL #000000
PIXEL 4 2 #FF0000
FRAME