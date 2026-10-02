package log

import (
	"strings"

	"go.uber.org/fx/fxevent"
)

type fxEventLogger struct {
	logger Logger
}

func NewFxEventLogger(l Logger) fxevent.Logger {
	return &fxEventLogger{logger: l}
}

func (f *fxEventLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		f.logger.Info("fx hook OnStart executing", "callee", e.FunctionName, "caller", e.CallerName)
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			f.logger.Error("fx hook OnStart failed", "callee", e.FunctionName, "caller", e.CallerName, Err(e.Err))
			return
		}
		f.logger.Info("fx hook OnStart executed", "callee", e.FunctionName, "caller", e.CallerName, "runtime", e.Runtime)
	case *fxevent.OnStopExecuting:
		f.logger.Info("fx hook OnStop executing", "callee", e.FunctionName, "caller", e.CallerName)
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			f.logger.Error("fx hook OnStop failed", "callee", e.FunctionName, "caller", e.CallerName, Err(e.Err))
			return
		}
		f.logger.Info("fx hook OnStop executed", "callee", e.FunctionName, "caller", e.CallerName, "runtime", e.Runtime)
	case *fxevent.Supplied:
		if e.Err != nil {
			f.logger.Error("fx supply failed", "type", e.TypeName, "module", e.ModuleName, Err(e.Err))
			return
		}
		f.logger.Debug("fx supplied", "type", e.TypeName, "module", e.ModuleName)
	case *fxevent.Provided:
		if e.Err != nil {
			f.logger.Error("fx provide failed", "constructor", e.ConstructorName, "module", e.ModuleName, Err(e.Err))
			return
		}
		f.logger.Debug("fx provided", "constructor", e.ConstructorName, "module", e.ModuleName, "types", strings.Join(e.OutputTypeNames, ","))
	case *fxevent.Replaced:
		if e.Err != nil {
			f.logger.Error("fx replace failed", "module", e.ModuleName, Err(e.Err))
			return
		}
		f.logger.Debug("fx replaced", "module", e.ModuleName, "types", strings.Join(e.OutputTypeNames, ","))
	case *fxevent.Decorated:
		if e.Err != nil {
			f.logger.Error("fx decorate failed", "decorator", e.DecoratorName, "module", e.ModuleName, Err(e.Err))
			return
		}
		f.logger.Debug("fx decorated", "decorator", e.DecoratorName, "module", e.ModuleName, "types", strings.Join(e.OutputTypeNames, ","))
	case *fxevent.BeforeRun:
		f.logger.Debug("fx before run", "name", e.Name, "kind", e.Kind, "module", e.ModuleName)
	case *fxevent.Run:
		if e.Err != nil {
			f.logger.Error("fx run failed", "name", e.Name, "kind", e.Kind, "module", e.ModuleName, Err(e.Err))
			return
		}
		f.logger.Debug("fx run", "name", e.Name, "kind", e.Kind, "module", e.ModuleName, "runtime", e.Runtime)
	case *fxevent.Invoking:
		f.logger.Debug("fx invoking", "function", e.FunctionName, "module", e.ModuleName)
	case *fxevent.Invoked:
		if e.Err != nil {
			f.logger.Error("fx invoke failed", "function", e.FunctionName, "module", e.ModuleName, "trace", e.Trace, Err(e.Err))
		}
	case *fxevent.Stopping:
		f.logger.Info("fx stopping", "signal", e.Signal)
	case *fxevent.Stopped:
		if e.Err != nil {
			f.logger.Error("fx stop failed", Err(e.Err))
			return
		}
		f.logger.Info("fx stopped")
	case *fxevent.RollingBack:
		f.logger.Error("fx start failed, rolling back", Err(e.StartErr))
	case *fxevent.RolledBack:
		if e.Err != nil {
			f.logger.Error("fx rollback failed", Err(e.Err))
		}
	case *fxevent.Started:
		if e.Err != nil {
			f.logger.Error("fx start failed", Err(e.Err))
			return
		}
		f.logger.Info("fx started")
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			f.logger.Error("fx logger init failed", Err(e.Err))
			return
		}
		f.logger.Debug("fx logger initialized", "constructor", e.ConstructorName)
	}
}
