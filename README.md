RPIO for Go
===========

This project provides a simple Go wrapper around the C library wrapped by the
RPio python library.  The purpose of this project is to make use of the
software PWM implementation used in the python library, in order to simplify
porting some robotics code that uses PWM on pins without hardware PWM support.

If you don't require software PWM, you are likely better off using the native
Go RPio implementation found at https://github.com/stianeikeland/go-rpio.

Additionally, this will likely not support all of the python functions. I am
focusing only on the ones I need for another project at the moment.

At the moment, this project only adds the `rpio.go` file, which uses cgo to
expose the RPio functions. The C and header files have only been modified in
minor ways, mostly to remove any usage of `Python.h`.

