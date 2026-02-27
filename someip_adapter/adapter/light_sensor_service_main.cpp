// SPDX-License-Identifier: MIT
// Copyright (c) 2026 MBition GmbH

#include <csignal>

#include <vsomeip/vsomeip.hpp>

#include "light_sensor_service.h"

LightSensorService *ls_srv_ptr(nullptr);

void handle_signal(int _signal) {
    if (ls_srv_ptr != nullptr &&
            (_signal == SIGINT || _signal == SIGTERM))
        ls_srv_ptr->stop();
}

int main(int argc, char **argv)
{
    (void)argc;
    (void)argv;

    LightSensorService ls_srv;

    ls_srv_ptr = &ls_srv;
    signal(SIGINT, handle_signal);
    signal(SIGTERM, handle_signal);

    if (ls_srv.init()) {
        ls_srv.start();

        return 0;
    } else {
        return 1;
    }
}
