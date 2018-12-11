package main

import "fmt"
import "encoding/hex"
import "log"
import "time"
import "github.com/stianeikeland/go-rpio"

//Make Exit on Ctrl +C
import "os"
import "os/signal"
import "syscall"

const (
        I2C_SLAVE = 0x0703
)

type Queue struct {
  size int
  value []float64
}

// I2Cstruc represents a connection to I2C-device.
type I2Cstruc struct {
        addr uint8
        bus  int
        rc   *os.File
}

func ioctl(fd, cmd, arg uintptr) error {
        _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, cmd, arg, 0, 0, 0)
        if err != 0 {
                return err
        }
        return nil
}

// NewI2C opens a connection for I2C-device.
// SMBus (System Management Bus) protocol over I2C
// supported as well: you should preliminary specify
// register address to read from, either write register
// together with the data in case of write operations.
func NewI2Cdevice(addr uint8, bus int) (*I2Cstruc, error) {
        f, err := os.OpenFile(fmt.Sprintf("/dev/i2c-%d", bus), os.O_RDWR, 0600)
        if err != nil {
                return nil, err
        }
        if err := ioctl(f.Fd(), I2C_SLAVE, uintptr(addr)); err != nil {
                return nil, err
        }
        v := &I2Cstruc{rc: f, bus: bus, addr: addr}
        return v, nil
}

// Write sends bytes to the remote I2C-device. The interpretation of
// the message is implementation-dependant.
func (v *I2Cstruc) WriteBytes(buf []byte) (int, error) {
        return v.write(buf)
}

func (v *I2Cstruc) write(buf []byte) (int, error) {
        return v.rc.Write(buf)
}

// Close I2C-connection.
func (v *I2Cstruc) Close() error {
        return v.rc.Close()
}

func (b *Queue) Pushtoqueue(first_element float64) float64 {
  sum:=first_element
  for i:=b.size-1; i>0; i-- {
    b.value[i]=b.value[i-1]
    sum+=b.value[i]
  }
  b.value[0]=first_element
return sum/float64(b.size)
}

func (b *Queue) Getqueueaverage() float64 {
  sum:=float64(0)
  for i:=0; i<b.size; i++ {
    sum+=b.value[i]
  }
return sum/float64(b.size)
}

func main () {
var average_hour,average_sec,average_min Queue
average_hour.size=24
average_sec.size=60
average_min.size=60
average_hour.value=make([]float64,average_hour.size)
average_sec.value=make([]float64,average_sec.size)
average_min.value=make([]float64,average_min.size)

err:=rpio.Open()
if err != nil {
  log.Fatal(err)
}
//GPIO 7 - eto 4 pin BCM
pin := rpio.Pin(4)
pin.Input()
pin.PullUp()
pin.Detect(rpio.FallEdge)

a,_ := NewI2Cdevice(0x19,1)
fmt.Println("Starting")

turnoff,err := hex.DecodeString("00")
if err != nil {
  log.Fatal(err)
}
turnon,err := hex.DecodeString("71")
if err != nil {
  log.Fatal(err)
}

var counter int
counter=0

a.WriteBytes(turnon)

tick_sec := time.Tick(1* time.Second)
tick_min := time.Tick(60* time.Second)
//tick_hour := time.Tick(3600* time.Second)

//Activating Ctrl+C
quit := make(chan os.Signal)
signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
cleanupDone := make(chan struct{})
go func() {
    <-quit
    close(cleanupDone)
}()
//Main CYCLE
for {
  select {
    case <-cleanupDone:
      pin.Detect(rpio.NoEdge)
      a.WriteBytes(turnoff)
      rpio.Close()
      a.Close()
      fmt.Println("Finished")
      os.Exit(1)
    case <-tick_sec:
      average_tick:=average_sec.Pushtoqueue(float64(counter))
      fmt.Printf("Impulses: %d. Average per second(\u00B5Sv per hour): %.3f. Average per minute(\u00B5Sv per hour): %.3f. Average per hour(\u00B5Sv per hour): %.5f.\n",counter , average_tick*.75,average_min.Pushtoqueue(average_tick)*.75,average_hour.Getqueueaverage()*.75)
      counter=0
    case <-tick_min:
      average_hour.Pushtoqueue(average_min.Getqueueaverage())
    default:
       if pin.EdgeDetected() {
          counter++
       }
//For STS-6 Maximum - 1000 counts per second. So sleep for a while
       time.Sleep(1 * time.Millisecond)
  }

}
}
