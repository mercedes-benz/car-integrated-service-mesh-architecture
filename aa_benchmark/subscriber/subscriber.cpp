// SPDX-License-Identifier: MIT
// Copyright (c) 2026 MBition GmbH

#include <chrono>
#include <csignal>
#include <cstdlib>
#include <thread>
#include <ara/log/logger.h>
#include <ara/core/initialization.h>
#include <fstream>

#include "ara/com/types.h"

#include "com/mercedes_benz/method_benchmark/aracm_method_benchmark_proxy.h"

const std::string METHOD_INSTANCE_ID = "someip:34343";

ara::core::Result<void> g_rt = ara::core::Initialize();

std::atomic<bool> g_rx_killswitch{false};

static void sigHandler(int f_sig)
{
    std::ignore = f_sig;

    g_rx_killswitch = true;
}

namespace it_usrdoc = ::com::mercedes_benz::method_benchmark;
namespace usrdoc
{
enum ESubscriberStates : uint8_t
{
    SM_WAIT_FOR_SERVICE = 0u,
    SM_METHOD_CALL
};

// Implementation of the actual benchmark
void MethodRx()
{
    ESubscriberStates l_current_state = SM_WAIT_FOR_SERVICE;

    ara::com::InstanceIdentifier l_instance(METHOD_INSTANCE_ID);

    std::shared_ptr<it_usrdoc::proxy::AraCM_Method_BenchmarkProxy> l_proxy;
    ara::core::Future<it_usrdoc::OutputMethodIO> l_future;

	// Initial discovery of service
    auto l_result = it_usrdoc::proxy::AraCM_Method_BenchmarkProxy::FindService(l_instance);
    while (true)
    {
    	if (!l_result.HasValue())
    	{
    		std::this_thread::sleep_for(std::chrono::milliseconds(2000));

    		l_result = it_usrdoc::proxy::AraCM_Method_BenchmarkProxy::FindService(l_instance);

    		continue;
    	}

    	for (auto l_item : l_result.Value())
    	{
    		l_proxy = std::make_shared<it_usrdoc::proxy::AraCM_Method_BenchmarkProxy>(l_item);

    		l_current_state = SM_METHOD_CALL;

    		break;
    	}

    	if (l_current_state == SM_METHOD_CALL)
    		break;

    	std::this_thread::sleep_for(std::chrono::milliseconds(2000));

    	l_result = it_usrdoc::proxy::AraCM_Method_BenchmarkProxy::FindService(l_instance);
    }

    std::ofstream l_fout("times.stats");

    auto l_start = std::chrono::high_resolution_clock::now();

    // Warmup for 10s
    while (std::chrono::duration<double, std::milli>(std::chrono::high_resolution_clock::now()-l_start).count() < 10000)
    {

		MsgRequest l_input;
		l_input.message = std::string("6MVjeMfskUeonWBkGUY5SOf49xcE1sVGlDAzBMudiCjYwZDYFO0IUmI4mKoPtukX43aj76XMkY1NcQGHZSNXv06dyvsXseKvPAm78mpUFI33cHK2xpMJJm4nT098tp1BiO9IlvV45KCUFcLHrvFNQ0DysQniNoVmMIFITAJhfkIbzaRP7nKkARjqcxdyh64XRiroJ9hJ6JiJnSh9681WWbLW2mEXBbf15mRb7zX2aiRvyMV4cSSvvIbNfaSTtqNmpnQ6fyTLsTtgXbUgOWEmxa2mEewGLUSvbQzVdV7WnbNfjSp8qdMChCAmToXVOV7iXfcC4X3EWSSK3CczJtQQo8aTIsI4bALQ3epHKNqVrNER76uFcqAYcMWoBMyHz3ehzGHPhxYlgrZ3xWi0hvqiQ8lU0q1pER19Y5kTZZyu69pC9grnNSicxnXB14YHAoECqzIC2GN7d16dQCEx3pNGo81dqerCAJTsxJJwlcxVFmxfSv4iLwTs8mkXZfewk8rlRtJhbKNbddkkAsDLX8DbQeT5qkGOwG84V6DuabLeVgyJnhkKVOuCJ7Ub57xZOqL6n5hh3CvX8Ii3O023Zcoef01BzryKcfx9x5fs79yzMH825gzC4kIrYl83nUy6vcEAkNi2dNlr9NHzL7rBQWX2zlhmTQYs0h74xTPFz8VzkNbOPJvBEbedwjVljkgWo7jqSMJq2wdhWSRucD2g2klVgKjpWtVuaKnHeA5xp7nAvw2x9SAVN1x3QNO5QvKoNwBZF5XcMvHJxGDZh4XSAhtPHEmh41WEIEODYVFuZ8ZpQLJ8gpBQQJ1XUI0EfDrZAKuJj22lv7NTaSYrKsUBBcFOoPqvbKVzBAZHURUL777kuv7kDayTfEeSjTfJGO6FCnkrMFaRNLKKMUlHZwa458bfnp3PyFmNLvemgQJmF4C325PhgiwxpRrE9GX0rOsXewsnJftGt5R8j8OJ0tIO9LxCeJXqSPbt3bD5NQzmYlJg");

		l_future = l_proxy->MethodIO(l_input);

		if (l_future.wait_for(std::chrono::milliseconds(100)) == ara::core::future_status::kReady)
		{
			ara::core::Result<it_usrdoc::OutputMethodIO, ara::core::ErrorCode> l_result = l_future.GetResult();
		}
    } 

    // Signal done
    std::this_thread::sleep_for(std::chrono::seconds(5));
    std::ofstream("wu_done");

	int l_j = 50; // gets replaced by the benchmarking runner
    for (int l_i=1;l_i<=l_j;++l_i)
    {
    	std::ignore = l_i;

    	l_start = std::chrono::high_resolution_clock::now();

    	l_result = it_usrdoc::proxy::AraCM_Method_BenchmarkProxy::FindService(l_instance);
		while (true)
		{
			if (!l_result.HasValue())
			{
				l_result = it_usrdoc::proxy::AraCM_Method_BenchmarkProxy::FindService(l_instance);

				continue;
			}

			for (auto l_item : l_result.Value())
			{
				l_proxy = std::make_shared<it_usrdoc::proxy::AraCM_Method_BenchmarkProxy>(l_item);

				l_current_state = SM_METHOD_CALL;

				break;
			}

			if (l_current_state == SM_METHOD_CALL)
				break;

			l_result = it_usrdoc::proxy::AraCM_Method_BenchmarkProxy::FindService(l_instance);
		}

    	MsgRequest l_input;
		l_input.message = std::string("6MVjeMfskUeonWBkGUY5SOf49xcE1sVGlDAzBMudiCjYwZDYFO0IUmI4mKoPtukX43aj76XMkY1NcQGHZSNXv06dyvsXseKvPAm78mpUFI33cHK2xpMJJm4nT098tp1BiO9IlvV45KCUFcLHrvFNQ0DysQniNoVmMIFITAJhfkIbzaRP7nKkARjqcxdyh64XRiroJ9hJ6JiJnSh9681WWbLW2mEXBbf15mRb7zX2aiRvyMV4cSSvvIbNfaSTtqNmpnQ6fyTLsTtgXbUgOWEmxa2mEewGLUSvbQzVdV7WnbNfjSp8qdMChCAmToXVOV7iXfcC4X3EWSSK3CczJtQQo8aTIsI4bALQ3epHKNqVrNER76uFcqAYcMWoBMyHz3ehzGHPhxYlgrZ3xWi0hvqiQ8lU0q1pER19Y5kTZZyu69pC9grnNSicxnXB14YHAoECqzIC2GN7d16dQCEx3pNGo81dqerCAJTsxJJwlcxVFmxfSv4iLwTs8mkXZfewk8rlRtJhbKNbddkkAsDLX8DbQeT5qkGOwG84V6DuabLeVgyJnhkKVOuCJ7Ub57xZOqL6n5hh3CvX8Ii3O023Zcoef01BzryKcfx9x5fs79yzMH825gzC4kIrYl83nUy6vcEAkNi2dNlr9NHzL7rBQWX2zlhmTQYs0h74xTPFz8VzkNbOPJvBEbedwjVljkgWo7jqSMJq2wdhWSRucD2g2klVgKjpWtVuaKnHeA5xp7nAvw2x9SAVN1x3QNO5QvKoNwBZF5XcMvHJxGDZh4XSAhtPHEmh41WEIEODYVFuZ8ZpQLJ8gpBQQJ1XUI0EfDrZAKuJj22lv7NTaSYrKsUBBcFOoPqvbKVzBAZHURUL777kuv7kDayTfEeSjTfJGO6FCnkrMFaRNLKKMUlHZwa458bfnp3PyFmNLvemgQJmF4C325PhgiwxpRrE9GX0rOsXewsnJftGt5R8j8OJ0tIO9LxCeJXqSPbt3bD5NQzmYlJg");

		l_future = l_proxy->MethodIO(l_input);

		if (l_future.wait_for(std::chrono::milliseconds(100)) == ara::core::future_status::kReady)
		{
			ara::core::Result<it_usrdoc::OutputMethodIO, ara::core::ErrorCode> l_result = l_future.GetResult();

			auto l_end = std::chrono::high_resolution_clock::now();
			l_fout << std::chrono::duration<double, std::milli>(l_end-l_start).count() << '\n';
		}
    }

    l_fout << std::flush;
}

}

int main(int argc, char* argv[])
{
    std::ignore = argc;
    std::ignore = argv;

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

    std::thread l_runner(usrdoc::MethodRx);
    l_runner.join();

    // Signal done
    std::ofstream("done");

    return (EXIT_SUCCESS);
}
