package render

/*
DO NOT EDIT!
This code was generated automatically using github.com/gojuno/minimock v1.9
The original interface "Ui" can be found in github.com/mitchellh/cli
*/
import (
	"sync/atomic"
	"time"

	"github.com/gojuno/minimock"
	testify_assert "github.com/stretchr/testify/assert"
)

//UiMock implements github.com/mitchellh/cli.Ui
type UiMock struct {
	t minimock.Tester

	AskFunc       func(p string) (r string, r1 error)
	AskCounter    uint64
	AskPreCounter uint64
	AskMock       mUiMockAsk

	AskSecretFunc       func(p string) (r string, r1 error)
	AskSecretCounter    uint64
	AskSecretPreCounter uint64
	AskSecretMock       mUiMockAskSecret

	ErrorFunc       func(p string)
	ErrorCounter    uint64
	ErrorPreCounter uint64
	ErrorMock       mUiMockError

	InfoFunc       func(p string)
	InfoCounter    uint64
	InfoPreCounter uint64
	InfoMock       mUiMockInfo

	OutputFunc       func(p string)
	OutputCounter    uint64
	OutputPreCounter uint64
	OutputMock       mUiMockOutput

	WarnFunc       func(p string)
	WarnCounter    uint64
	WarnPreCounter uint64
	WarnMock       mUiMockWarn
}

//NewUiMock returns a mock for github.com/mitchellh/cli.Ui
func NewUiMock(t minimock.Tester) *UiMock {
	m := &UiMock{t: t}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.AskMock = mUiMockAsk{mock: m}
	m.AskSecretMock = mUiMockAskSecret{mock: m}
	m.ErrorMock = mUiMockError{mock: m}
	m.InfoMock = mUiMockInfo{mock: m}
	m.OutputMock = mUiMockOutput{mock: m}
	m.WarnMock = mUiMockWarn{mock: m}

	return m
}

type mUiMockAsk struct {
	mock             *UiMock
	mockExpectations *UiMockAskParams
}

//UiMockAskParams represents input parameters of the Ui.Ask
type UiMockAskParams struct {
	p string
}

//Expect sets up expected params for the Ui.Ask
func (m *mUiMockAsk) Expect(p string) *mUiMockAsk {
	m.mockExpectations = &UiMockAskParams{p}
	return m
}

//Return sets up a mock for Ui.Ask to return Return's arguments
func (m *mUiMockAsk) Return(r string, r1 error) *UiMock {
	m.mock.AskFunc = func(p string) (string, error) {
		return r, r1
	}
	return m.mock
}

//Set uses given function f as a mock of Ui.Ask method
func (m *mUiMockAsk) Set(f func(p string) (r string, r1 error)) *UiMock {
	m.mock.AskFunc = f
	return m.mock
}

//Ask implements github.com/mitchellh/cli.Ui interface
func (m *UiMock) Ask(p string) (r string, r1 error) {
	atomic.AddUint64(&m.AskPreCounter, 1)
	defer atomic.AddUint64(&m.AskCounter, 1)

	if m.AskMock.mockExpectations != nil {
		testify_assert.Equal(m.t, *m.AskMock.mockExpectations, UiMockAskParams{p},
			"Ui.Ask got unexpected parameters")

		if m.AskFunc == nil {

			m.t.Fatal("No results are set for the UiMock.Ask")

			return
		}
	}

	if m.AskFunc == nil {
		m.t.Fatal("Unexpected call to UiMock.Ask")
		return
	}

	return m.AskFunc(p)
}

//AskMinimockCounter returns a count of UiMock.AskFunc invocations
func (m *UiMock) AskMinimockCounter() uint64 {
	return atomic.LoadUint64(&m.AskCounter)
}

//AskMinimockPreCounter returns the value of UiMock.Ask invocations
func (m *UiMock) AskMinimockPreCounter() uint64 {
	return atomic.LoadUint64(&m.AskPreCounter)
}

type mUiMockAskSecret struct {
	mock             *UiMock
	mockExpectations *UiMockAskSecretParams
}

//UiMockAskSecretParams represents input parameters of the Ui.AskSecret
type UiMockAskSecretParams struct {
	p string
}

//Expect sets up expected params for the Ui.AskSecret
func (m *mUiMockAskSecret) Expect(p string) *mUiMockAskSecret {
	m.mockExpectations = &UiMockAskSecretParams{p}
	return m
}

//Return sets up a mock for Ui.AskSecret to return Return's arguments
func (m *mUiMockAskSecret) Return(r string, r1 error) *UiMock {
	m.mock.AskSecretFunc = func(p string) (string, error) {
		return r, r1
	}
	return m.mock
}

//Set uses given function f as a mock of Ui.AskSecret method
func (m *mUiMockAskSecret) Set(f func(p string) (r string, r1 error)) *UiMock {
	m.mock.AskSecretFunc = f
	return m.mock
}

//AskSecret implements github.com/mitchellh/cli.Ui interface
func (m *UiMock) AskSecret(p string) (r string, r1 error) {
	atomic.AddUint64(&m.AskSecretPreCounter, 1)
	defer atomic.AddUint64(&m.AskSecretCounter, 1)

	if m.AskSecretMock.mockExpectations != nil {
		testify_assert.Equal(m.t, *m.AskSecretMock.mockExpectations, UiMockAskSecretParams{p},
			"Ui.AskSecret got unexpected parameters")

		if m.AskSecretFunc == nil {

			m.t.Fatal("No results are set for the UiMock.AskSecret")

			return
		}
	}

	if m.AskSecretFunc == nil {
		m.t.Fatal("Unexpected call to UiMock.AskSecret")
		return
	}

	return m.AskSecretFunc(p)
}

//AskSecretMinimockCounter returns a count of UiMock.AskSecretFunc invocations
func (m *UiMock) AskSecretMinimockCounter() uint64 {
	return atomic.LoadUint64(&m.AskSecretCounter)
}

//AskSecretMinimockPreCounter returns the value of UiMock.AskSecret invocations
func (m *UiMock) AskSecretMinimockPreCounter() uint64 {
	return atomic.LoadUint64(&m.AskSecretPreCounter)
}

type mUiMockError struct {
	mock             *UiMock
	mockExpectations *UiMockErrorParams
}

//UiMockErrorParams represents input parameters of the Ui.Error
type UiMockErrorParams struct {
	p string
}

//Expect sets up expected params for the Ui.Error
func (m *mUiMockError) Expect(p string) *mUiMockError {
	m.mockExpectations = &UiMockErrorParams{p}
	return m
}

//Return sets up a mock for Ui.Error to return Return's arguments
func (m *mUiMockError) Return() *UiMock {
	m.mock.ErrorFunc = func(p string) {
		return
	}
	return m.mock
}

//Set uses given function f as a mock of Ui.Error method
func (m *mUiMockError) Set(f func(p string)) *UiMock {
	m.mock.ErrorFunc = f
	return m.mock
}

//Error implements github.com/mitchellh/cli.Ui interface
func (m *UiMock) Error(p string) {
	atomic.AddUint64(&m.ErrorPreCounter, 1)
	defer atomic.AddUint64(&m.ErrorCounter, 1)

	if m.ErrorMock.mockExpectations != nil {
		testify_assert.Equal(m.t, *m.ErrorMock.mockExpectations, UiMockErrorParams{p},
			"Ui.Error got unexpected parameters")

		if m.ErrorFunc == nil {

			m.t.Fatal("No results are set for the UiMock.Error")

			return
		}
	}

	if m.ErrorFunc == nil {
		m.t.Fatal("Unexpected call to UiMock.Error")
		return
	}

	m.ErrorFunc(p)
}

//ErrorMinimockCounter returns a count of UiMock.ErrorFunc invocations
func (m *UiMock) ErrorMinimockCounter() uint64 {
	return atomic.LoadUint64(&m.ErrorCounter)
}

//ErrorMinimockPreCounter returns the value of UiMock.Error invocations
func (m *UiMock) ErrorMinimockPreCounter() uint64 {
	return atomic.LoadUint64(&m.ErrorPreCounter)
}

type mUiMockInfo struct {
	mock             *UiMock
	mockExpectations *UiMockInfoParams
}

//UiMockInfoParams represents input parameters of the Ui.Info
type UiMockInfoParams struct {
	p string
}

//Expect sets up expected params for the Ui.Info
func (m *mUiMockInfo) Expect(p string) *mUiMockInfo {
	m.mockExpectations = &UiMockInfoParams{p}
	return m
}

//Return sets up a mock for Ui.Info to return Return's arguments
func (m *mUiMockInfo) Return() *UiMock {
	m.mock.InfoFunc = func(p string) {
		return
	}
	return m.mock
}

//Set uses given function f as a mock of Ui.Info method
func (m *mUiMockInfo) Set(f func(p string)) *UiMock {
	m.mock.InfoFunc = f
	return m.mock
}

//Info implements github.com/mitchellh/cli.Ui interface
func (m *UiMock) Info(p string) {
	atomic.AddUint64(&m.InfoPreCounter, 1)
	defer atomic.AddUint64(&m.InfoCounter, 1)

	if m.InfoMock.mockExpectations != nil {
		testify_assert.Equal(m.t, *m.InfoMock.mockExpectations, UiMockInfoParams{p},
			"Ui.Info got unexpected parameters")

		if m.InfoFunc == nil {

			m.t.Fatal("No results are set for the UiMock.Info")

			return
		}
	}

	if m.InfoFunc == nil {
		m.t.Fatal("Unexpected call to UiMock.Info")
		return
	}

	m.InfoFunc(p)
}

//InfoMinimockCounter returns a count of UiMock.InfoFunc invocations
func (m *UiMock) InfoMinimockCounter() uint64 {
	return atomic.LoadUint64(&m.InfoCounter)
}

//InfoMinimockPreCounter returns the value of UiMock.Info invocations
func (m *UiMock) InfoMinimockPreCounter() uint64 {
	return atomic.LoadUint64(&m.InfoPreCounter)
}

type mUiMockOutput struct {
	mock             *UiMock
	mockExpectations *UiMockOutputParams
}

//UiMockOutputParams represents input parameters of the Ui.Output
type UiMockOutputParams struct {
	p string
}

//Expect sets up expected params for the Ui.Output
func (m *mUiMockOutput) Expect(p string) *mUiMockOutput {
	m.mockExpectations = &UiMockOutputParams{p}
	return m
}

//Return sets up a mock for Ui.Output to return Return's arguments
func (m *mUiMockOutput) Return() *UiMock {
	m.mock.OutputFunc = func(p string) {
		return
	}
	return m.mock
}

//Set uses given function f as a mock of Ui.Output method
func (m *mUiMockOutput) Set(f func(p string)) *UiMock {
	m.mock.OutputFunc = f
	return m.mock
}

//Output implements github.com/mitchellh/cli.Ui interface
func (m *UiMock) Output(p string) {
	atomic.AddUint64(&m.OutputPreCounter, 1)
	defer atomic.AddUint64(&m.OutputCounter, 1)

	if m.OutputMock.mockExpectations != nil {
		testify_assert.Equal(m.t, *m.OutputMock.mockExpectations, UiMockOutputParams{p},
			"Ui.Output got unexpected parameters")

		if m.OutputFunc == nil {

			m.t.Fatal("No results are set for the UiMock.Output")

			return
		}
	}

	if m.OutputFunc == nil {
		m.t.Fatal("Unexpected call to UiMock.Output")
		return
	}

	m.OutputFunc(p)
}

//OutputMinimockCounter returns a count of UiMock.OutputFunc invocations
func (m *UiMock) OutputMinimockCounter() uint64 {
	return atomic.LoadUint64(&m.OutputCounter)
}

//OutputMinimockPreCounter returns the value of UiMock.Output invocations
func (m *UiMock) OutputMinimockPreCounter() uint64 {
	return atomic.LoadUint64(&m.OutputPreCounter)
}

type mUiMockWarn struct {
	mock             *UiMock
	mockExpectations *UiMockWarnParams
}

//UiMockWarnParams represents input parameters of the Ui.Warn
type UiMockWarnParams struct {
	p string
}

//Expect sets up expected params for the Ui.Warn
func (m *mUiMockWarn) Expect(p string) *mUiMockWarn {
	m.mockExpectations = &UiMockWarnParams{p}
	return m
}

//Return sets up a mock for Ui.Warn to return Return's arguments
func (m *mUiMockWarn) Return() *UiMock {
	m.mock.WarnFunc = func(p string) {
		return
	}
	return m.mock
}

//Set uses given function f as a mock of Ui.Warn method
func (m *mUiMockWarn) Set(f func(p string)) *UiMock {
	m.mock.WarnFunc = f
	return m.mock
}

//Warn implements github.com/mitchellh/cli.Ui interface
func (m *UiMock) Warn(p string) {
	atomic.AddUint64(&m.WarnPreCounter, 1)
	defer atomic.AddUint64(&m.WarnCounter, 1)

	if m.WarnMock.mockExpectations != nil {
		testify_assert.Equal(m.t, *m.WarnMock.mockExpectations, UiMockWarnParams{p},
			"Ui.Warn got unexpected parameters")

		if m.WarnFunc == nil {

			m.t.Fatal("No results are set for the UiMock.Warn")

			return
		}
	}

	if m.WarnFunc == nil {
		m.t.Fatal("Unexpected call to UiMock.Warn")
		return
	}

	m.WarnFunc(p)
}

//WarnMinimockCounter returns a count of UiMock.WarnFunc invocations
func (m *UiMock) WarnMinimockCounter() uint64 {
	return atomic.LoadUint64(&m.WarnCounter)
}

//WarnMinimockPreCounter returns the value of UiMock.Warn invocations
func (m *UiMock) WarnMinimockPreCounter() uint64 {
	return atomic.LoadUint64(&m.WarnPreCounter)
}

//ValidateCallCounters checks that all mocked methods of the interface have been called at least once
//Deprecated: please use MinimockFinish method or use Finish method of minimock.Controller
func (m *UiMock) ValidateCallCounters() {

	if m.AskFunc != nil && atomic.LoadUint64(&m.AskCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.Ask")
	}

	if m.AskSecretFunc != nil && atomic.LoadUint64(&m.AskSecretCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.AskSecret")
	}

	if m.ErrorFunc != nil && atomic.LoadUint64(&m.ErrorCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.Error")
	}

	if m.InfoFunc != nil && atomic.LoadUint64(&m.InfoCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.Info")
	}

	if m.OutputFunc != nil && atomic.LoadUint64(&m.OutputCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.Output")
	}

	if m.WarnFunc != nil && atomic.LoadUint64(&m.WarnCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.Warn")
	}

}

//CheckMocksCalled checks that all mocked methods of the interface have been called at least once
//Deprecated: please use MinimockFinish method or use Finish method of minimock.Controller
func (m *UiMock) CheckMocksCalled() {
	m.Finish()
}

//Finish checks that all mocked methods of the interface have been called at least once
//Deprecated: please use MinimockFinish or use Finish method of minimock.Controller
func (m *UiMock) Finish() {
	m.MinimockFinish()
}

//MinimockFinish checks that all mocked methods of the interface have been called at least once
func (m *UiMock) MinimockFinish() {

	if m.AskFunc != nil && atomic.LoadUint64(&m.AskCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.Ask")
	}

	if m.AskSecretFunc != nil && atomic.LoadUint64(&m.AskSecretCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.AskSecret")
	}

	if m.ErrorFunc != nil && atomic.LoadUint64(&m.ErrorCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.Error")
	}

	if m.InfoFunc != nil && atomic.LoadUint64(&m.InfoCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.Info")
	}

	if m.OutputFunc != nil && atomic.LoadUint64(&m.OutputCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.Output")
	}

	if m.WarnFunc != nil && atomic.LoadUint64(&m.WarnCounter) == 0 {
		m.t.Fatal("Expected call to UiMock.Warn")
	}

}

//Wait waits for all mocked methods to be called at least once
//Deprecated: please use MinimockWait or use Wait method of minimock.Controller
func (m *UiMock) Wait(timeout time.Duration) {
	m.MinimockWait(timeout)
}

//MinimockWait waits for all mocked methods to be called at least once
//this method is called by minimock.Controller
func (m *UiMock) MinimockWait(timeout time.Duration) {
	timeoutCh := time.After(timeout)
	for {
		ok := true
		ok = ok && (m.AskFunc == nil || atomic.LoadUint64(&m.AskCounter) > 0)
		ok = ok && (m.AskSecretFunc == nil || atomic.LoadUint64(&m.AskSecretCounter) > 0)
		ok = ok && (m.ErrorFunc == nil || atomic.LoadUint64(&m.ErrorCounter) > 0)
		ok = ok && (m.InfoFunc == nil || atomic.LoadUint64(&m.InfoCounter) > 0)
		ok = ok && (m.OutputFunc == nil || atomic.LoadUint64(&m.OutputCounter) > 0)
		ok = ok && (m.WarnFunc == nil || atomic.LoadUint64(&m.WarnCounter) > 0)

		if ok {
			return
		}

		select {
		case <-timeoutCh:

			if m.AskFunc != nil && atomic.LoadUint64(&m.AskCounter) == 0 {
				m.t.Error("Expected call to UiMock.Ask")
			}

			if m.AskSecretFunc != nil && atomic.LoadUint64(&m.AskSecretCounter) == 0 {
				m.t.Error("Expected call to UiMock.AskSecret")
			}

			if m.ErrorFunc != nil && atomic.LoadUint64(&m.ErrorCounter) == 0 {
				m.t.Error("Expected call to UiMock.Error")
			}

			if m.InfoFunc != nil && atomic.LoadUint64(&m.InfoCounter) == 0 {
				m.t.Error("Expected call to UiMock.Info")
			}

			if m.OutputFunc != nil && atomic.LoadUint64(&m.OutputCounter) == 0 {
				m.t.Error("Expected call to UiMock.Output")
			}

			if m.WarnFunc != nil && atomic.LoadUint64(&m.WarnCounter) == 0 {
				m.t.Error("Expected call to UiMock.Warn")
			}

			m.t.Fatalf("Some mocks were not called on time: %s", timeout)
			return
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

//AllMocksCalled returns true if all mocked methods were called before the execution of AllMocksCalled,
//it can be used with assert/require, i.e. assert.True(mock.AllMocksCalled())
func (m *UiMock) AllMocksCalled() bool {

	if m.AskFunc != nil && atomic.LoadUint64(&m.AskCounter) == 0 {
		return false
	}

	if m.AskSecretFunc != nil && atomic.LoadUint64(&m.AskSecretCounter) == 0 {
		return false
	}

	if m.ErrorFunc != nil && atomic.LoadUint64(&m.ErrorCounter) == 0 {
		return false
	}

	if m.InfoFunc != nil && atomic.LoadUint64(&m.InfoCounter) == 0 {
		return false
	}

	if m.OutputFunc != nil && atomic.LoadUint64(&m.OutputCounter) == 0 {
		return false
	}

	if m.WarnFunc != nil && atomic.LoadUint64(&m.WarnCounter) == 0 {
		return false
	}

	return true
}
