package client

import (
	"flag"
	"os"
	"runtime/pprof"

	"github.com/mcluseau/kube-proxy2/pkg/api/localnetv1"
	"k8s.io/klog"
)

type HandlerFunc func(items []*localnetv1.ServiceEndpoints)

func Default() (epc *EndpointsClient, once bool, nodeName string, stop func()) {
	onceFlag := flag.Bool("once", false, "only one fetch loop")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	flag.StringVar(&nodeName, "node-name", "", "node name to request to the proxy server")

	epc = New(flag.CommandLine)

	flag.Parse()

	once = *onceFlag

	if *cpuprofile == "" {
		stop = func() {}
	} else {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			klog.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		stop = pprof.StopCPUProfile
	}

	epc.CancelOnSignals()

	if nodeName == "" {
		var err error
		nodeName, err = os.Hostname()
		if err != nil {
			klog.Fatal("no node-name set and hostname request failed: ", err)
		}
	}

	return
}

// Run the client with the standard options
func Run(req *localnetv1.WatchReq, handlers ...HandlerFunc) {
	epc, once, nodeName, stop := Default()
	defer stop()

	if req == nil {
		req = &localnetv1.WatchReq{}
	}
	if req.NodeName == "" {
		req.NodeName = nodeName
	}

	for {
		items, canceled := epc.Next(req)

		if canceled {
			klog.Infof("finished")
			return
		}

		for _, handler := range handlers {
			handler(items)
		}

		if once {
			return
		}
	}
}

// RunWithIterator runs the client with the standard options, using the iterated version of Next.
// It should consume less memory as the dataset is processed as it's read instead of buffered.
// The handler MUST check iter.Err to ensure the dataset was fuly retrieved without error.
func RunWithIterator(req *localnetv1.WatchReq, handler func(*Iterator)) {
	epc, once, nodeName, stop := Default()
	defer stop()

	if req == nil {
		req = &localnetv1.WatchReq{}
	}
	if req.NodeName == "" {
		req.NodeName = nodeName
	}

	for {
		iter := epc.NextIterator(req)

		if iter.Canceled {
			klog.Infof("finished")
			return
		}

		handler(iter)

		if iter.RecvErr != nil {
			klog.Error("recv error: ", iter.RecvErr)
		}

		if once {
			return
		}
	}
}
