package hardware

/*
#cgo LDFLAGS: -ldxgi -luuid
#include <windows.h>
#include <dxgi.h>
#include <string.h>

#define MAX_GPUS 8
#define MAX_GPU_NAME_LEN 256

typedef struct {
    int count;
    char names[MAX_GPUS][MAX_GPU_NAME_LEN];
} GPUEnumResult;

int GetMonitorGpuIndex(int monitorIndex) {
    IDXGIFactory1* factory = NULL;
    if (CreateDXGIFactory1(&IID_IDXGIFactory1, (void**)&factory) != S_OK)
        return -1;

    IDXGIAdapter1* adapter = NULL;
    IDXGIOutput* output = NULL;
    int currentMonitor = 0;

    for (UINT a = 0; factory->lpVtbl->EnumAdapters1(factory, a, &adapter) != DXGI_ERROR_NOT_FOUND; a++) {
        for (UINT o = 0; adapter->lpVtbl->EnumOutputs(adapter, o, &output) != DXGI_ERROR_NOT_FOUND; o++) {
            if (currentMonitor == monitorIndex) {
                output->lpVtbl->Release(output);
                adapter->lpVtbl->Release(adapter);
                factory->lpVtbl->Release(factory);
                return a;
            }
            output->lpVtbl->Release(output);
            currentMonitor++;
        }
        adapter->lpVtbl->Release(adapter);
    }

    factory->lpVtbl->Release(factory);
    return -1;
}

GPUEnumResult EnumerateGPUs() {
    GPUEnumResult result;
    memset(&result, 0, sizeof(result));

    IDXGIFactory1* factory = NULL;
    if (CreateDXGIFactory1(&IID_IDXGIFactory1, (void**)&factory) != S_OK)
        return result;

    IDXGIAdapter1* adapter = NULL;
    for (UINT i = 0; i < MAX_GPUS && factory->lpVtbl->EnumAdapters1(factory, i, &adapter) != DXGI_ERROR_NOT_FOUND; i++) {
        DXGI_ADAPTER_DESC1 desc;
        if (adapter->lpVtbl->GetDesc1(adapter, &desc) == S_OK) {
            WideCharToMultiByte(CP_UTF8, 0, desc.Description, -1, result.names[result.count], MAX_GPU_NAME_LEN, NULL, NULL);
            result.count++;
        }
        adapter->lpVtbl->Release(adapter);
    }

    factory->lpVtbl->Release(factory);
    return result;
}
*/
import "C"

func GetMonitorGPUIndex(monitorIndex int) int {
	return int(C.GetMonitorGpuIndex(C.int(monitorIndex)))
}

func EnumerateGPUsDXGI() []string {
	result := C.EnumerateGPUs()
	var gpuNames []string
	for i := 0; i < int(result.count); i++ {
		name := C.GoString(&result.names[i][0])
		if name != "" {
			gpuNames = append(gpuNames, name)
		}
	}
	return gpuNames
}
