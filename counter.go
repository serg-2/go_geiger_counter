package main

import "github.com/d2r2/go-i2c"
//Changed in go-i2c writeBytes - turned off logger
//Changed in go-logger 320 string - turned off logger on Ctrl+C
//Changed in go-logger 318 string - no lg needed
//Changed in go-logger 319 string - no Options needed
//Changed in go-shell 56 - no log needed
import "fmt"
import "encoding/hex"
import "log"
import "time"
import "github.com/stianeikeland/go-rpio"

//Make Ctrl +C
import "os"
import "os/signal"
import "syscall"

type Queue struct {
  size int
  value []float64
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

a,_ := i2c.NewI2C(0x19,1)
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
