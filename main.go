package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	fmt.Println("what is today's lucky number?")

	c := make(chan int)
	go getLuckyNum(c)

	num := <-c

	fmt.Println("Today's lucky number is", num)

	close(c)
}

func getLuckyNum(c chan<- int) {
	fmt.Println("...")

	// 占いにかかる時間はランダム
	rand.Seed(time.Now().Unix())
	time.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond)

	num := rand.Intn(10)
	c <- num
}
