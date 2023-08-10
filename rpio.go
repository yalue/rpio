// This is a package providing GPIO access to the raspberry pi from Go. Wraps
// the C code used by the python RPio.GPIO library.
package rpio

import (
	"fmt"
)

/*
#include "common.h"
#include "cpuinfo.h"
#include "event_gpio.h"
#include "c_gpio.h"
#include "soft_pwm.h"

int GetGPIODirection(int gpio) {
	return gpio_direction[gpio];
}

int SetGPIODirection(int gpio, int direction) {
	gpio_direction[gpio] = direction;
}

void SetGPIOMode(int mode) {
	gpio_mode = mode;
}

int GetGPIOMode(void) {
	return gpio_mode;
}
*/
import "C"

const (
	PUD_OFF      = C.PUD_OFF
	PUD_DOWN     = C.PUD_DOWN
	PUD_UP       = C.PUD_UP
	HIGH         = C.HIGH
	INPUT        = C.INPUT
	OUTPUT       = C.OUTPUT
	ALT0         = C.ALT0
	BOARD        = C.BOARD
	BCM          = C.BCM
	MODE_UNKNOWN = C.MODE_UNKNOWN
)

// Set to true if Setup has been called successfully.
var setupOK bool

// Returns a non-nil error if the package's Setup() function either failed or
// hasn't been called.
func SetupOK() error {
	if setupOK {
		return nil
	}
	return fmt.Errorf("The package's Setup() function either failed or " +
		"has not been called")
}

// Must be called prior to using any other GPIO functionality. Returns an error
// if one occurs. Returns nil if Setup has already been completed.
func Setup() error {
	if setupOK {
		return nil
	}
	for i := 0; i < 54; i++ {
		C.SetGPIODirection(C.int(i), -1)
	}
	result := C.setup()
	if result == C.SETUP_OK {
		setupOK = true
		return nil
	}
	if result == C.SETUP_DEVMEM_FAIL {
		return fmt.Errorf("No access to /dev/mem. Try running as root")
	}
	if result == C.SETUP_MALLOC_FAIL {
		return fmt.Errorf("setup() failed: malloc failure")
	}
	if result == C.SETUP_CPUINFO_FAIL {
		return fmt.Errorf("Unable to open /proc/cpuinfo")
	}
	if result == C.SETUP_NO_PERI_ADDR {
		return fmt.Errorf("Unable to determine SOC peripheral base address")
	}
	return fmt.Errorf("Error setting up GPIO: setup() returned %d", result)
}

// Cleans up GPIO activity on the given channel.
func CleanupChannel(channel uint8) error {
	e := SetupOK()
	if e != nil {
		return e
	}
	gpio, e := GetGPIONumber(channel)
	if e != nil {
		return fmt.Errorf("Bad channel number (%d): %w", channel, e)
	}
	if C.GetGPIODirection(C.int(gpio)) == -1 {
		return nil
	}
	C.setup_gpio(C.int(gpio), INPUT, PUD_OFF)
	C.SetGPIODirection(C.int(gpio), -1)
	return nil
}

// May be called to clean up the GPIO functionality when it's no longer needed.
// Returns an error if Setup() hasn't been called yet.
func Cleanup() error {
	if !setupOK {
		return fmt.Errorf("The library isn't set up")
	}
	// Used in py_gpio.c to reset all channels during cleanup.
	for i := 0; i < 54; i++ {
		if C.GetGPIODirection(C.int(i)) == -1 {
			continue
		}
		C.setup_gpio(C.int(i), INPUT, PUD_OFF)
		C.SetGPIODirection(C.int(i), -1)
	}
	C.cleanup()
	setupOK = false
	return nil
}

// Returns a non-nil error if stuff hasn't been set up properly.
func CheckGPIOPriv() error {
	result := C.check_gpio_priv()
	if result != 0 {
		return fmt.Errorf("check_gpio_priv() returned %d, expected 0", result)
	}
	return nil
}

func GetGPIONumber(channel uint8) (uint8, error) {
	value := C.uint(0)
	result := C.get_gpio_number(C.int(channel), &value)
	if result != 0 {
		return 0, fmt.Errorf("get_gpio_number() failed with code %d", result)
	}
	if value >= 0xff {
		return 0, fmt.Errorf("channel %d's gpio value (%d) overflows uint8",
			channel, value)
	}
	return uint8(value), nil
}

func GPIOFunction(gpio uint8) int {
	return int(C.gpio_function(C.int(gpio)))
}

// Returns a non-nil error if the argument isn't PUD_OFF, PUD_UP, or PUD_DOWN.
func validatePullUpDown(pullUpDown uint8) error {
	if (pullUpDown == PUD_OFF) || (pullUpDown == PUD_UP) ||
		(pullUpDown == PUD_DOWN) {
		return nil
	}
	return fmt.Errorf("Invalid pull up/down value: %d", pullUpDown)
}

// Sets up a GPIO channel. The channel must be a RPi pin number of BCM number
// depending on mode. The output parameter must be true to set the pin to
// output mode or false if used as an input. pullUpDown must be PUD_OFF,
// PUD_UP, or PUD_DOWN.
func SetupChannel(channel, pullUpDown uint8, output bool) error {
	e := SetupOK()
	if e != nil {
		return e
	}
	e = validatePullUpDown(pullUpDown)
	if e != nil {
		return e
	}
	gpio, e := GetGPIONumber(channel)
	if e != nil {
		return fmt.Errorf("Bad channel (%d): %w", channel, e)
	}
	f := GPIOFunction(gpio)
	if (f != 0) && (f != 1) {
		return fmt.Errorf("Channel %d already in use", channel)
	}
	directionInt := C.GetGPIODirection(C.int(gpio))
	if (f == 1) && (directionInt != -1) {
		return fmt.Errorf("Channel %d already in use", channel)
	}
	directionInt = C.int(INPUT)
	if output {
		directionInt = OUTPUT
		pullUpDown = PUD_OFF
	}
	C.setup_gpio(C.int(gpio), directionInt, C.int(pullUpDown))
	return nil
}

// Sets the given channel to high or low, depending on whether the high arg
// is true. Returns an error if one occurs, including if the channel is not
// in output mode.
func OutputGPIO(channel uint8, high bool) error {
	e := SetupOK()
	if e != nil {
		return e
	}
	gpio, e := GetGPIONumber(channel)
	if e != nil {
		return fmt.Errorf("Bad channel (%d): %w", channel, e)
	}
	if C.GetGPIODirection(C.int(gpio)) != OUTPUT {
		return fmt.Errorf("Channel %d isn't set as an output", channel)
	}
	value := C.int(0)
	if high {
		value = 1
	}
	C.output_gpio(C.int(gpio), value)
	return nil
}

// Returns true if the given channel's input is HIGH and false if it's LOW.
// Returns an error if the channel is invalid.
func InputGPIO(channel uint8) (bool, error) {
	e := SetupOK()
	if e != nil {
		return false, e
	}
	gpio, e := GetGPIONumber(channel)
	if e != nil {
		return false, fmt.Errorf("Bad channel (%d): %w", channel, e)
	}
	dir := C.GetGPIODirection(C.int(gpio))
	if (dir != INPUT) && (dir != OUTPUT) {
		return false, fmt.Errorf("Channel %d isn't set up", channel)
	}
	return (C.input_gpio(C.int(gpio)) != 0), nil
}

// Sets the GPIO mode. The new mode must be either BOARD or BCM.
func SetMode(newMode int) error {
	e := SetupOK()
	if e != nil {
		return e
	}
	oldMode := C.GetGPIOMode()
	if (oldMode != MODE_UNKNOWN) && (oldMode != C.int(newMode)) {
		return fmt.Errorf("A different mode has already been set")
	}
	if (newMode != BOARD) && (newMode != BCM) {
		return fmt.Errorf("Invalid new mode (%d), must be either BOARD or BCM",
			newMode)
	}
	C.SetGPIOMode(C.int(newMode))
	return nil
}

// Returns the current GPIO mode. This may be either MODE_UNKNOWN, MODE_BOARD,
// or MODE_BCM. Returns an error if the package's Setup() function hasn't been
// called yet.
func GetMode() (int, error) {
	e := SetupOK()
	if e != nil {
		return MODE_UNKNOWN, e
	}
	return int(C.GetGPIOMode()), nil
}

// Returns true if there is a PWM for this pin, otherwise returns false.
func PWMExists(channel uint8) (bool, error) {
	e := SetupOK()
	if e != nil {
		return false, e
	}
	gpio, e := GetGPIONumber(channel)
	if e != nil {
		return false, fmt.Errorf("Bad channel (%d): %w", channel, e)
	}
	result := C.pwm_exists(C.uint(gpio))
	return (result != 0), nil
}

// Wraps the state for a software PWM output.
type PWMObject struct {
	gpio      uint8
	frequency float32
	dutyCycle float32
}

// Sets up the given channel to use soft PWM. The channel must already be in
// output mode. Corresponds to PWM.__init__ in python.
func NewPWM(channel uint8, frequency float32) (*PWMObject, error) {
	gpio, e := GetGPIONumber(channel)
	if e != nil {
		return nil, e
	}
	pwmExists, e := PWMExists(channel)
	if e != nil {
		return nil, e
	}
	if pwmExists {
		return nil, fmt.Errorf("A PWM already exists for GPIO channel %d",
			channel)
	}
	if C.GetGPIODirection(C.int(gpio)) != OUTPUT {
		return nil, fmt.Errorf("The GPIO channel %d must be set up as an "+
			"output for PWM", channel)
	}
	if frequency <= 0.0 {
		return nil, fmt.Errorf("PWM frequency must be positive, got %f",
			frequency)
	}
	return &PWMObject{
		gpio:      gpio,
		frequency: frequency,
		dutyCycle: 0.0,
	}, nil
}

// Starts the software GPIO. Corresponds to PWM.start in python. Duty cycle is
// a *percentage* (up to 100.0).
func (p *PWMObject) Start(dutyCycle float32) error {
	if (dutyCycle < 0.0) || (dutyCycle > 100.0) {
		return fmt.Errorf("Duty cycle must be between 0 and 100. Got %f",
			dutyCycle)
	}
	p.dutyCycle = dutyCycle
	C.pwm_set_duty_cycle(C.uint(p.gpio), C.float(p.dutyCycle))
	C.pwm_start(C.uint(p.gpio))
	return nil
}

// As with PWMObject.Start, the dutyCycle here must be a percentage.
func (p *PWMObject) ChangeDutyCycle(dutyCycle float32) error {
	if (dutyCycle < 0.0) || (dutyCycle > 100.0) {
		return fmt.Errorf("Duty cycle must be between 0 and 100. Got %f",
			dutyCycle)
	}
	p.dutyCycle = dutyCycle
	C.pwm_set_duty_cycle(C.uint(p.gpio), C.float(p.dutyCycle))
	return nil
}

func (p *PWMObject) ChangeFrequency(frequency float32) error {
	if frequency <= 0.0 {
		return fmt.Errorf("Frequency must be positive; got %f", frequency)
	}
	p.frequency = frequency
	C.pwm_set_frequency(C.uint(p.gpio), C.float(p.frequency))
	return nil
}

// Stops the PWM running.
func (p *PWMObject) Stop() error {
	C.pwm_stop(C.uint(p.gpio))
	return nil
}

// TODO: Wrap event-related functions, i.e. edge-detect, etc from py_gpio.c
