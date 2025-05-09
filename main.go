package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func workerFileCounter(dirsToCount chan string, fileCounts chan int, wg *sync.WaitGroup, directoriesBeingProcessed *sync.WaitGroup) {
	defer wg.Done()
	for dir := range dirsToCount {
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Printf("Error on %v: %v\n", dir, err)
		}
		var count int
		for _, entry := range entries {
			if !entry.IsDir() {
				count += 1
			} else {
				count += 0
			}
		}
		fileCounts <- count
		directoriesBeingProcessed.Done()
	}
}

func workerFileFinder(dirsToCount chan string, dirsToScan chan string, wg *sync.WaitGroup, directoriesBeingProcessed *sync.WaitGroup) {
	defer wg.Done()
	for dir := range dirsToScan {
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Printf("Error on %v: %v\n", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				directoriesBeingProcessed.Add(2)
				dirsToCount <- filepath.Join(dir, entry.Name())
				dirsToScan <- filepath.Join(dir, entry.Name())
			}
		}
		directoriesBeingProcessed.Done()
	}
}

func main() {
	dirsToScan := make(chan string, 1000)
	dirsToCount := make(chan string, 1000)
	fileCounts := make(chan int, 1000)

	var counterWorkers sync.WaitGroup
	var finderWorkers sync.WaitGroup
	var directoriesBeingProcessed sync.WaitGroup
	var countingWg sync.WaitGroup

	countingWg.Add(1)
	directoriesBeingProcessed.Add(1)

	totalFiles := 0
	go func() {
		defer countingWg.Done()
		for count := range fileCounts {
			totalFiles += count
		}
	}()

	go func() {
		directoriesBeingProcessed.Wait()
		close(dirsToScan)
		close(dirsToCount)
	}()

	numCounterWorkers := 5
	numFinderWorkers := 5

	for i := 0; i < numCounterWorkers; i++ {
		counterWorkers.Add(1)
		go workerFileCounter(dirsToCount, fileCounts, &counterWorkers, &directoriesBeingProcessed)
	}
	for i := 0; i < numFinderWorkers; i++ {
		finderWorkers.Add(1)
		go workerFileFinder(dirsToCount, dirsToScan, &finderWorkers, &directoriesBeingProcessed)
	}

	dirsToScan <- "."

	finderWorkers.Wait()
	counterWorkers.Wait()

	close(fileCounts)

	countingWg.Wait()
	fmt.Printf("Total files: %d\n", totalFiles)
}
