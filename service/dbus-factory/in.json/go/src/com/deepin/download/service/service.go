/*This file is auto generate by pkg.linuxdeepin.com/dbus-generator. Don't edit it*/
package service

import "pkg.linuxdeepin.com/lib/dbus"
import "pkg.linuxdeepin.com/lib/dbus/property"
import "reflect"
import "sync"
import "runtime"
import "fmt"
import "errors"

/*prevent compile error*/
var _ = fmt.Println
var _ = runtime.SetFinalizer
var _ = sync.NewCond
var _ = reflect.TypeOf
var _ = property.BaseObserver{}

type Service struct {
	Path     dbus.ObjectPath
	DestName string
	core     *dbus.Object

	signals       map[chan *dbus.Signal]bool
	signalsLocker sync.Mutex
}

func (obj Service) _createSignalChan() chan *dbus.Signal {
	obj.signalsLocker.Lock()
	ch := make(chan *dbus.Signal, 30)
	getBus().Signal(ch)
	obj.signals[ch] = false
	obj.signalsLocker.Unlock()
	return ch
}
func (obj Service) _deleteSignalChan(ch chan *dbus.Signal) {
	obj.signalsLocker.Lock()
	delete(obj.signals, ch)
	getBus().DetachSignal(ch)
	close(ch)
	obj.signalsLocker.Unlock()
}
func DestroyService(obj *Service) {
	obj.signalsLocker.Lock()
	for ch, _ := range obj.signals {
		getBus().DetachSignal(ch)
		close(ch)
	}
	obj.signals = make(map[chan *dbus.Signal]bool)
	obj.signalsLocker.Unlock()

}

func (obj Service) AddTask(arg0 string, arg1 []string, arg2 []int64, arg3 []string, arg4 string) (arg5 string, _err error) {
	_err = obj.core.Call("com.deepin.download.service.AddTask", 0, arg0, arg1, arg2, arg3, arg4).Store(&arg5)
	if _err != nil {
		fmt.Println(_err)
	}
	return
}

func (obj Service) PauseTask(arg0 string) (_err error) {
	_err = obj.core.Call("com.deepin.download.service.PauseTask", 0, arg0).Store()
	if _err != nil {
		fmt.Println(_err)
	}
	return
}

func (obj Service) ResumeTask(arg0 string) (_err error) {
	_err = obj.core.Call("com.deepin.download.service.ResumeTask", 0, arg0).Store()
	if _err != nil {
		fmt.Println(_err)
	}
	return
}

func (obj Service) StopTask(arg0 string) (_err error) {
	_err = obj.core.Call("com.deepin.download.service.StopTask", 0, arg0).Store()
	if _err != nil {
		fmt.Println(_err)
	}
	return
}

func (obj Service) ConnectStart(callback func(arg0 string)) func() {
	__conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',path='"+string(obj.Path)+"', interface='com.deepin.download.service',sender='"+obj.DestName+"',member='Start'")
	sigChan := obj._createSignalChan()
	go func() {
		for v := range sigChan {
			if v.Path != obj.Path || v.Name != "com.deepin.download.service.Start" || 1 != len(v.Body) {
				continue
			}
			if reflect.TypeOf(v.Body[0]) != reflect.TypeOf((*string)(nil)).Elem() {
				continue
			}

			callback(v.Body[0].(string))
		}
	}()
	return func() {
		obj._deleteSignalChan(sigChan)
	}
}

func (obj Service) ConnectUpdate(callback func(arg0 string, arg1 int32, arg2 int32, arg3 int32, arg4 int32, arg5 int64, arg6 int64)) func() {
	__conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',path='"+string(obj.Path)+"', interface='com.deepin.download.service',sender='"+obj.DestName+"',member='Update'")
	sigChan := obj._createSignalChan()
	go func() {
		for v := range sigChan {
			if v.Path != obj.Path || v.Name != "com.deepin.download.service.Update" || 7 != len(v.Body) {
				continue
			}
			if reflect.TypeOf(v.Body[0]) != reflect.TypeOf((*string)(nil)).Elem() {
				continue
			}
			if reflect.TypeOf(v.Body[1]) != reflect.TypeOf((*int32)(nil)).Elem() {
				continue
			}
			if reflect.TypeOf(v.Body[2]) != reflect.TypeOf((*int32)(nil)).Elem() {
				continue
			}
			if reflect.TypeOf(v.Body[3]) != reflect.TypeOf((*int32)(nil)).Elem() {
				continue
			}
			if reflect.TypeOf(v.Body[4]) != reflect.TypeOf((*int32)(nil)).Elem() {
				continue
			}
			if reflect.TypeOf(v.Body[5]) != reflect.TypeOf((*int64)(nil)).Elem() {
				continue
			}
			if reflect.TypeOf(v.Body[6]) != reflect.TypeOf((*int64)(nil)).Elem() {
				continue
			}

			callback(v.Body[0].(string), v.Body[1].(int32), v.Body[2].(int32), v.Body[3].(int32), v.Body[4].(int32), v.Body[5].(int64), v.Body[6].(int64))
		}
	}()
	return func() {
		obj._deleteSignalChan(sigChan)
	}
}

func (obj Service) ConnectFinish(callback func(arg0 string)) func() {
	__conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',path='"+string(obj.Path)+"', interface='com.deepin.download.service',sender='"+obj.DestName+"',member='Finish'")
	sigChan := obj._createSignalChan()
	go func() {
		for v := range sigChan {
			if v.Path != obj.Path || v.Name != "com.deepin.download.service.Finish" || 1 != len(v.Body) {
				continue
			}
			if reflect.TypeOf(v.Body[0]) != reflect.TypeOf((*string)(nil)).Elem() {
				continue
			}

			callback(v.Body[0].(string))
		}
	}()
	return func() {
		obj._deleteSignalChan(sigChan)
	}
}

func (obj Service) ConnectPause(callback func(arg0 string)) func() {
	__conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',path='"+string(obj.Path)+"', interface='com.deepin.download.service',sender='"+obj.DestName+"',member='Pause'")
	sigChan := obj._createSignalChan()
	go func() {
		for v := range sigChan {
			if v.Path != obj.Path || v.Name != "com.deepin.download.service.Pause" || 1 != len(v.Body) {
				continue
			}
			if reflect.TypeOf(v.Body[0]) != reflect.TypeOf((*string)(nil)).Elem() {
				continue
			}

			callback(v.Body[0].(string))
		}
	}()
	return func() {
		obj._deleteSignalChan(sigChan)
	}
}

func (obj Service) ConnectStop(callback func(arg0 string)) func() {
	__conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',path='"+string(obj.Path)+"', interface='com.deepin.download.service',sender='"+obj.DestName+"',member='Stop'")
	sigChan := obj._createSignalChan()
	go func() {
		for v := range sigChan {
			if v.Path != obj.Path || v.Name != "com.deepin.download.service.Stop" || 1 != len(v.Body) {
				continue
			}
			if reflect.TypeOf(v.Body[0]) != reflect.TypeOf((*string)(nil)).Elem() {
				continue
			}

			callback(v.Body[0].(string))
		}
	}()
	return func() {
		obj._deleteSignalChan(sigChan)
	}
}

func (obj Service) ConnectError(callback func(arg0 string, arg1 int32, arg2 string)) func() {
	__conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',path='"+string(obj.Path)+"', interface='com.deepin.download.service',sender='"+obj.DestName+"',member='Error'")
	sigChan := obj._createSignalChan()
	go func() {
		for v := range sigChan {
			if v.Path != obj.Path || v.Name != "com.deepin.download.service.Error" || 3 != len(v.Body) {
				continue
			}
			if reflect.TypeOf(v.Body[0]) != reflect.TypeOf((*string)(nil)).Elem() {
				continue
			}
			if reflect.TypeOf(v.Body[1]) != reflect.TypeOf((*int32)(nil)).Elem() {
				continue
			}
			if reflect.TypeOf(v.Body[2]) != reflect.TypeOf((*string)(nil)).Elem() {
				continue
			}

			callback(v.Body[0].(string), v.Body[1].(int32), v.Body[2].(string))
		}
	}()
	return func() {
		obj._deleteSignalChan(sigChan)
	}
}

func (obj Service) ConnectResume(callback func(arg0 string)) func() {
	__conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',path='"+string(obj.Path)+"', interface='com.deepin.download.service',sender='"+obj.DestName+"',member='Resume'")
	sigChan := obj._createSignalChan()
	go func() {
		for v := range sigChan {
			if v.Path != obj.Path || v.Name != "com.deepin.download.service.Resume" || 1 != len(v.Body) {
				continue
			}
			if reflect.TypeOf(v.Body[0]) != reflect.TypeOf((*string)(nil)).Elem() {
				continue
			}

			callback(v.Body[0].(string))
		}
	}()
	return func() {
		obj._deleteSignalChan(sigChan)
	}
}

func NewService(destName string, path dbus.ObjectPath) (*Service, error) {
	if !path.IsValid() {
		return nil, errors.New("The path of '" + string(path) + "' is invalid.")
	}

	core := getBus().Object(destName, path)

	obj := &Service{Path: path, DestName: destName, core: core, signals: make(map[chan *dbus.Signal]bool)}

	runtime.SetFinalizer(obj, func(_obj *Service) { DestroyService(_obj) })
	return obj, nil
}
