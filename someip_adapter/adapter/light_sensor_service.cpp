
// SPDX-License-Identifier: MIT
// Copyright (c) 2026 MBition GmbH
 
#include "light_sensor_service.h"

#include <chrono>
#include <functional>
#include <set>
#include <string>

bool
LightSensorService::init()
{
    if (!app_->init()) {
        LOG_ERR("Couldn't initialize application");

        return false;
    }

    app_->register_message_handler(service_id, service_instance_id, service_method_id,
            std::bind(&LightSensorService::on_message_cbk, this, std::placeholders::_1));
    app_->register_state_handler(std::bind(&LightSensorService::on_state_cbk, this, std::placeholders::_1));

    return true;
}

void
LightSensorService::start()
{
    app_->start();
}

void
LightSensorService::stop()
{   
    app_->stop_offer_service(service_id, service_instance_id);

    app_->unregister_state_handler();

    app_->unregister_message_handler(service_id, service_instance_id, service_method_id);

    app_->clear_all_handler();

    app_->stop();
}

void
LightSensorService::on_state_cbk(vsomeip::state_type_e _state)
{
    if(_state == vsomeip::state_type_e::ST_REGISTERED)
    {
        app_->offer_service(service_id, service_instance_id);
    }
}

void
LightSensorService::on_message_cbk(const std::shared_ptr<vsomeip::message> &_request)
{
    std::shared_ptr<vsomeip::message> resp = rtm_->create_response(_request);

    std::string str;
    if(is_day_)
    {
        str += "Day";
    }
    else
    {
        str += "Night";
    }

    std::shared_ptr<vsomeip::payload> resp_pl = rtm_->create_payload();
    std::vector<vsomeip::byte_t> pl_data(str.begin(), str.end());
    resp_pl->set_data(pl_data);

    resp->set_payload(resp_pl);

    app_->send(resp);
}