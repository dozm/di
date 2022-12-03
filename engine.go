package di

type ServiceAccessor func(*ContainerEngineScope) (any, error)

type ContainerEngine interface {
	RealizeService(CallSite) (ServiceAccessor, error)
}

type containerEngine struct {
	container *container
}

func (engine *containerEngine) RealizeService(callSite CallSite) (ServiceAccessor, error) {
	return func(scope *ContainerEngineScope) (any, error) {
		return CallSiteResolverInstance.Resolve(callSite, scope)
	}, nil
}

// func (engine *containerEngine) RealizeService(callSite CallSite) (ServiceAccessor, error) {
// 	callCount := uint32(0)

// 	return func(scope *ContainerEngineScope) (any, error) {
// 		result, err := CallSiteResolverInstance.Resolve(callSite, scope)
// 		if callCount < 2 && atomic.AddUint32(&callCount, 1) == 2 {
// 			go func(sp *container) {
// 				// TODO: replace service accessor
// 				// sp.ReplaceServiceAccessor(accessor)
// 			}(engine.container)
// 		}

// 		return result, err
// 	}, nil
// }

func newContainerEngine(c *container) ContainerEngine {
	return &containerEngine{container: c}
}
