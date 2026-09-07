package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"java-app/pkg/jvm"
)

func main() {
	fmt.Println("==================================================================")
	fmt.Println("  EMPIRICAL CHALLENGER 2 VERIFICATION SUITE — MILESTONE 3 (JVM)   ")
	fmt.Println("==================================================================")

	passed := true

	// 1. JVM Initialization & Performance Benchmark
	if err := testJVMInitializationPerformance(); err != nil {
		fmt.Printf("[FAIL] TestJVMInitializationPerformance: %v\n", err)
		passed = false
	} else {
		fmt.Println("[PASS] TestJVMInitializationPerformance")
	}

	// 2. Thread Safety & Concurrency Test
	if err := testJVMThreadSafetyAndConcurrency(); err != nil {
		fmt.Printf("[FAIL] TestJVMThreadSafetyAndConcurrency: %v\n", err)
		passed = false
	} else {
		fmt.Println("[PASS] TestJVMThreadSafetyAndConcurrency")
	}

	// 3. Signal Handling (-Xrs) Test
	if err := testSignalHandlingXrs(); err != nil {
		fmt.Printf("[FAIL] TestSignalHandlingXrs: %v\n", err)
		passed = false
	} else {
		fmt.Println("[PASS] TestSignalHandlingXrs")
	}

	// 4. Binary Build, LDD Dynamic Link, and Runtime Execution Test
	if err := testBinaryExecutionAndDynamicLinking(); err != nil {
		fmt.Printf("[FAIL] TestBinaryExecutionAndDynamicLinking: %v\n", err)
		passed = false
	} else {
		fmt.Println("[PASS] TestBinaryExecutionAndDynamicLinking")
	}

	// 5. Run Full Package Unit Test Suite
	if err := testFullPackageTestSuite(); err != nil {
		fmt.Printf("[FAIL] TestFullPackageTestSuite: %v\n", err)
		passed = false
	} else {
		fmt.Println("[PASS] TestFullPackageTestSuite")
	}

	fmt.Println("==================================================================")
	if passed {
		fmt.Println("  VERDICT: ALL EMPIRICAL CHALLENGER TESTS PASSED (APPROVE)       ")
		fmt.Println("==================================================================")
		os.Exit(0)
	} else {
		fmt.Println("  VERDICT: EMPIRICAL CHALLENGER TESTS FAILED (REJECT)            ")
		fmt.Println("==================================================================")
		os.Exit(1)
	}
}

func testJVMInitializationPerformance() error {
	runtime.LockOSThread()

	var mBefore runtime.MemStats
	runtime.ReadMemStats(&mBefore)
	start := time.Now()

	vm, err := jvm.InitJVM("./j2me-stubs/bin")
	duration := time.Since(start)

	var mAfter runtime.MemStats
	runtime.ReadMemStats(&mAfter)

	if err != nil {
		return fmt.Errorf("InitJVM failed: %w", err)
	}
	if vm == nil {
		return fmt.Errorf("InitJVM returned nil JVM instance")
	}

	fmt.Printf("  • JVM initialization duration: %v (%d ms)\n", duration, duration.Milliseconds())
	fmt.Printf("  • Heap Alloc delta: %d KB\n", (mAfter.Alloc-mBefore.Alloc)/1024)
	fmt.Printf("  • Sys Memory delta: %d KB\n", (mAfter.Sys-mBefore.Sys)/1024)

	// Test subsequent idempotent call
	subsequentStart := time.Now()
	vm2, err2 := jvm.InitJVM("./j2me-stubs/bin")
	subsequentDuration := time.Since(subsequentStart)

	if err2 != nil {
		return fmt.Errorf("subsequent InitJVM failed: %w", err2)
	}
	if vm2 != vm {
		return fmt.Errorf("expected identical JVM instance pointer %p, got %p", vm, vm2)
	}
	fmt.Printf("  • Idempotent InitJVM call duration: %v (%d µs)\n", subsequentDuration, subsequentDuration.Microseconds())

	return nil
}

func testJVMThreadSafetyAndConcurrency() error {
	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errChan := make(chan error, goroutines)
	jvmChan := make(chan *jvm.JVM, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			inst, err := jvm.InitJVM("./j2me-stubs/bin")
			if err != nil {
				errChan <- fmt.Errorf("goroutine %d InitJVM error: %w", id, err)
				return
			}
			if inst == nil {
				errChan <- fmt.Errorf("goroutine %d returned nil JVM", id)
				return
			}

			global := jvm.GetJVM()
			if global != inst {
				errChan <- fmt.Errorf("goroutine %d GetJVM mismatch: got %p, want %p", id, global, inst)
				return
			}

			if inst.GetEnv() == nil || inst.GetVM() == nil || inst.Env() == nil || inst.VM() == nil {
				errChan <- fmt.Errorf("goroutine %d returned nil JNI pointer", id)
				return
			}

			jvmChan <- inst
		}(i)
	}

	wg.Wait()
	close(errChan)
	close(jvmChan)

	var firstErr error
	for err := range errChan {
		if firstErr == nil {
			firstErr = err
		}
		fmt.Printf("  • Thread safety error: %v\n", err)
	}
	if firstErr != nil {
		return firstErr
	}

	var firstInst *jvm.JVM
	count := 0
	for inst := range jvmChan {
		count++
		if firstInst == nil {
			firstInst = inst
		} else if inst != firstInst {
			return fmt.Errorf("inconsistent JVM pointer: expected %p, got %p", firstInst, inst)
		}
	}
	fmt.Printf("  • %d concurrent goroutines successfully retrieved identical JVM pointer %p\n", count, firstInst)
	return nil
}

func testSignalHandlingXrs() error {
	runtime.LockOSThread()

	_, err := jvm.InitJVM("./j2me-stubs/bin")
	if err != nil {
		return fmt.Errorf("InitJVM failed: %w", err)
	}

	signalsToTest := []syscall.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP,
		syscall.SIGUSR1,
	}

	for _, sig := range signalsToTest {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, sig)

		pid := os.Getpid()
		if err := syscall.Kill(pid, sig); err != nil {
			signal.Stop(sigChan)
			return fmt.Errorf("failed to send signal %v: %w", sig, err)
		}

		select {
		case caught := <-sigChan:
			if caught != sig {
				signal.Stop(sigChan)
				return fmt.Errorf("expected signal %v, got %v", sig, caught)
			}
			fmt.Printf("  • Caught signal %v in Go signal handler while JVM is active (-Xrs verified)\n", sig)
		case <-time.After(2 * time.Second):
			signal.Stop(sigChan)
			return fmt.Errorf("timeout waiting for signal %v receipt in Go handler", sig)
		}
		signal.Stop(sigChan)
	}

	return nil
}

func testBinaryExecutionAndDynamicLinking() error {
	// Build binary
	buildCmd := exec.Command("go", "build", "-o", "j2mego_challenger2_bin", ".")
	buildCmd.Dir = "."
	var buildErrBuf bytes.Buffer
	buildCmd.Stderr = &buildErrBuf
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %v, stderr: %s", err, buildErrBuf.String())
	}
	defer os.Remove("j2mego_challenger2_bin")

	fmt.Println("  • go build -o j2mego succeeded")

	// LDD inspection
	lddCmd := exec.Command("ldd", "j2mego_challenger2_bin")
	lddOut, err := lddCmd.Output()
	if err != nil {
		return fmt.Errorf("ldd failed: %w", err)
	}

	lddStr := string(lddOut)
	if !strings.Contains(lddStr, "libjvm.so") {
		return fmt.Errorf("ldd output missing libjvm.so linkage: %s", lddStr)
	}
	if strings.Contains(lddStr, "libjvm.so => not found") {
		return fmt.Errorf("libjvm.so is not found by dynamic linker: %s", lddStr)
	}

	line := findLibjvmLine(lddStr)
	fmt.Printf("  • ldd dynamic link verified: %s\n", line)

	// Execute binary
	runCmd := exec.Command("timeout", "3", "./j2mego_challenger2_bin")
	var runOutBuf, runErrBuf bytes.Buffer
	runCmd.Stdout = &runOutBuf
	runCmd.Stderr = &runErrBuf

	_ = runCmd.Run()

	combined := runOutBuf.String() + "\n" + runErrBuf.String()
	if !strings.Contains(combined, "JVM initialized successfully!") {
		return fmt.Errorf("binary output missing 'JVM initialized successfully!'. Output:\n%s", combined)
	}

	fmt.Println("  • Binary execution printed 'JVM initialized successfully!'")
	return nil
}

func testFullPackageTestSuite() error {
	cmd := exec.Command("go", "test", "-count=1", "./...")
	cmd.Dir = "."
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("go test ./... failed: %v\nStdout:\n%s\nStderr:\n%s", err, outBuf.String(), errBuf.String())
	}
	fmt.Println("  • All package unit tests passed (100% pass rate)")
	return nil
}

func findLibjvmLine(lddOutput string) string {
	lines := strings.Split(lddOutput, "\n")
	for _, l := range lines {
		if strings.Contains(l, "libjvm.so") {
			return strings.TrimSpace(l)
		}
	}
	return ""
}
