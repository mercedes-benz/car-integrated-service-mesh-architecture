// SPDX-License-Identifier: MIT
// Copyright (c) 2026 MBition GmbH

#include <csignal>

#include <vsomeip/vsomeip.hpp>

#include "adapter.h"

Adapter *csa_ptr(nullptr);

void handle_signal(int _signal) {
    if (csa_ptr != nullptr &&
            (_signal == SIGINT || _signal == SIGTERM))
        csa_ptr->stop();
}

int main(int argc, char **argv)
{
    (void)argc;
    (void)argv;

    Adapter csa;

    csa_ptr = &csa;
    signal(SIGINT, handle_signal);
    signal(SIGTERM, handle_signal);

    if (csa.init()) {
        csa.start();

        return 0;
    } else {
        return 1;
    }
}
