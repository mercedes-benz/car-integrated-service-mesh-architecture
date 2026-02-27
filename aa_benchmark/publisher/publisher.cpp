// SPDX-License-Identifier: MIT
// Copyright (c) 2026 MBition GmbH

#include <chrono>
#include <csignal>
#include <thread>
#include <ara/log/logger.h>
#include <ara/core/initialization.h>

#include "ara/com/types.h"

#include "com/mercedes_benz/method_benchmark/aracm_method_benchmark_skeleton.h"

const std::string METHOD_INSTANCE_ID = "someip:34343";

ara::core::Result<void> g_rt = ara::core::Initialize();

// Signal handling
std::atomic<bool> g_tx_killswitch{false};

static void sigHandler(int f_sig)
{
    std::ignore = f_sig;

    g_tx_killswitch = true;
}

class Benchmark : public com::mercedes_benz::method_benchmark::skeleton::AraCM_Method_BenchmarkSkeleton
{
public:
    ~Benchmark()
    {
    }
    Benchmark(const ara::com::InstanceIdentifier& f_instance)
        : com::mercedes_benz::method_benchmark::skeleton::AraCM_Method_BenchmarkSkeleton(f_instance)
    {
    }

    ara::core::Future<com::mercedes_benz::method_benchmark::OutputMethodIO> MethodIO(const MsgRequest f_input)
    {
        MsgResponse l_response;
        l_response.message = std::move(f_input.message);

        struct com::mercedes_benz::method_benchmark::OutputMethodIO l_output = {};
        l_output.m_value_out = l_response;

        ara::core::Promise<com::mercedes_benz::method_benchmark::OutputMethodIO> l_promise;
        l_promise.set_value(l_output);

        return l_promise.get_future();
    }
};

void MethodTx()
{
    ara::com::InstanceIdentifier l_instance(METHOD_INSTANCE_ID);

    Benchmark l_service(l_instance);

    l_service.OfferService();
    
    while (false == g_tx_killswitch)
    {
        std::this_thread::sleep_for(std::chrono::milliseconds(60000));
    }

    l_service.StopOfferService();
}

int main()
{
    struct sigaction l_act;
    sigemptyset(&l_act.sa_mask);
    l_act.sa_handler = sigHandler;
    l_act.sa_flags   = 0;

    if (-1 == sigaction(SIGINT, &l_act, NULL))
    {
        exit(EXIT_FAILURE);
    }

    if (-1 == sigaction(SIGTERM, &l_act, NULL))
    {
        exit(EXIT_FAILURE);
    }

    std::thread l_runner(MethodTx);
    l_runner.join();

    return (EXIT_SUCCESS);
}
