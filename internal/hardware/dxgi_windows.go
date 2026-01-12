package hardware

/*
#cgo LDFLAGS: -ldxgi -luuid
#include <windows.h>
#include <dxgi.h>
#include <stdio.h>

int GetMonitorGpuIndex(int monitorIndex) {
    IDXGIFactory1* factory = NULL;
    if (CreateDXGIFactory1(&IID_IDXGIFactory1, (void**)&factory) != S_OK)
        return -1;

    IDXGIAdapter1* adapter = NULL;
    IDXGIOutput* output = NULL;

    int currentMonitor = 0;

    for (UINT a = 0;
         factory->lpVtbl->EnumAdapters1(factory, a, &adapter) != DXGI_ERROR_NOT_FOUND;
         a++) {

        for (UINT o = 0;
             adapter->lpVtbl->EnumOutputs(adapter, o, &output) != DXGI_ERROR_NOT_FOUND;
             o++) {

            DXGI_OUTPUT_DESC desc;
            output->lpVtbl->GetDesc(output, &desc);

            if (currentMonitor == monitorIndex) {
                output->lpVtbl->Release(output);
                adapter->lpVtbl->Release(adapter);
                factory->lpVtbl->Release(factory);
                return a; // GPU index
            }

            output->lpVtbl->Release(output);
            currentMonitor++;
        }

        adapter->lpVtbl->Release(adapter);
    }

    factory->lpVtbl->Release(factory);
    return -1;
}
*/
import "C"

// GetMonitorGPUIndex returns the index of the GPU that is driving the monitor at the given index.
// It returns -1 if the GPU/Monitor could not be found.
func GetMonitorGPUIndex(monitorIndex int) int {
	return int(C.GetMonitorGpuIndex(C.int(monitorIndex)))
}
