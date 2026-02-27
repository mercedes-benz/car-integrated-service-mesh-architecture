// SPDX-License-Identifier: MIT
// Copyright (c) 2026 MBition GmbH

#ifndef INCLUDED_LIGHT_SENSOR_SERVICE_H
#define INCLUDED_LIGHT_SENSOR_SERVICE_H

// ---- STL ----
#include <cstdio>
#include <memory>

// ---- VSOMEIP ----
#include <vsomeip/vsomeip.hpp>

#define LOG_INF(...) fprintf(stdout, __VA_ARGS__), fprintf(stdout, "\n")
#define LOG_ERR(...) fprintf(stderr, __VA_ARGS__), fprintf(stderr, "\n")

static vsomeip::service_t service_id = 0x1111;
static vsomeip::instance_t service_instance_id = 0x2222;
static vsomeip::method_t service_method_id = 0x3333;

class LightSensorService {
public:
    LightSensorService()
        : rtm_(vsomeip::runtime::get())
        , app_(rtm_->create_application())
        , is_day_(true)
    {}
    bool init();
    void start();
    void stop();
    void on_state_cbk(vsomeip::state_type_e _state);
    void on_message_cbk(const std::shared_ptr<vsomeip::message> &_request);
private:
    std::shared_ptr<vsomeip::runtime> rtm_;
    std::shared_ptr<vsomeip::application> app_;

    bool is_day_;
};

#endif // INCLUDED_LIGHT_SENSOR_SERVICE_H