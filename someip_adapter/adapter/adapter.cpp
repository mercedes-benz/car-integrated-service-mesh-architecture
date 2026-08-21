// SPDX-License-Identifier: MIT
// Copyright (c) 2026 MBition GmbH

#include "adapter.h"

#include <algorithm>
#include <limits>
#include <iostream>

bool
Adapter::init()
{
    if (!app_->init())
    {
        LOG_ERR("Couldn't initialize application");

        return false;
    }

    app_->register_state_handler(std::bind(&Adapter::on_state_cbk, this, std::placeholders::_1));
    app_->register_availability_handler(vsomeip::ANY_SERVICE, vsomeip::ANY_INSTANCE,
            std::bind(&Adapter::on_availability_cbk, this, std::placeholders::_1, std::placeholders::_2, std::placeholders::_3));

    
    return true;
}

void
Adapter::start()
{
    std::string server_address("0.0.0.0:50042");

    ServerBuilder builder;
    builder.AddListeningPort(server_address, grpc::InsecureServerCredentials());
    builder.RegisterService(&service_);

    cq_ = builder.AddCompletionQueue();
    server_ = builder.BuildAndStart();
    worker_ = std::make_unique<std::thread>([&]{ HandleRpcs(); });

    app_->start();
}

void
Adapter::stop()
{
    done_.store(true);

    server_->Shutdown();
    cq_->Shutdown();

    worker_->join();

    app_->unregister_state_handler();

    app_->clear_all_handler();

    app_->release_service(vsomeip::ANY_SERVICE, vsomeip::ANY_INSTANCE);
    
    app_->stop();
}

void
Adapter::CallData::Proceed()
{
    if(status_ == CREATE)
    {
        status_ = PROCESS;

        if(diff_)
        {
            service_->RequestGetAvailableServices(&ctx_, &request_, &responder_, cq_, cq_, this);
        }
        else
        {
            service_->RequestCallMethod(&ctx_, &request_2_, &responder_2_, cq_, cq_, this);
        }
    }
    else if(status_ == PROCESS)
    {
        new CallData(adapter_, service_, cq_, diff_);

        if(diff_)
        {
            for(auto const &instance : adapter_.services_)
            {
                Service* service = reply_.add_services();
                service->set_service_id(std::get<0>(instance));
                service->set_instance_id(std::get<1>(instance));
            }

            responder_.Finish(reply_, Status::OK, this);

            status_ = FINISH;
        }
        else
        {
            auto check_identifier = [&](const auto &_a, auto &_t)
            { 
                if(_a >= 0 && _a <= std::numeric_limits<uint16_t>::max())
                {
                    _t = _a;

                    return true;
                }

                return false;
            };

            if(!check_identifier(request_2_.service().service_id(), _sid) ||
               !check_identifier(request_2_.service().instance_id(), _iid) ||
               !check_identifier(request_2_.method_id(), _mid))
            {
                responder_2_.Finish(reply_2_, grpc::Status(grpc::StatusCode::INVALID_ARGUMENT, "invalid identifier provided"), this);
                
                status_ = FINISH;
                
                return;
            }

            auto found_instance = std::find(adapter_.services_.begin(), adapter_.services_.end(), std::make_pair(_sid, _iid));
            if(found_instance == adapter_.services_.end())
            {
                responder_2_.Finish(reply_2_, grpc::Status(grpc::StatusCode::INVALID_ARGUMENT, "provided service is unavailable"), this);
                
                status_ = FINISH;
                
                return;
            }

            adapter_.app_->register_message_handler(_sid, _iid, _mid, std::bind(&CallData::on_message_cbk, this, std::placeholders::_1));

            std::shared_ptr<vsomeip::message> rq = adapter_.rtm_->create_request();

            rq->set_service(_sid);
            rq->set_instance(_iid);
            rq->set_method(_mid);

            auto pl = adapter_.rtm_->create_payload();
            std::vector<vsomeip::byte_t> pl_data(request_2_.data().begin(), request_2_.data().end());

            pl->set_data(pl_data);
            rq->set_payload(pl);

            adapter_.app_->send(rq);

            session_ = rq->get_session();
        }
    } else {
        CHECK(status_ == FINISH);

        adapter_.app_->unregister_message_handler(_sid, _iid, _mid, this);

        delete this;
    }
}

void
Adapter::CallData::on_message_cbk(const std::shared_ptr<vsomeip::message> &_response)
{
    if(session_ == _response->get_session())
    {
        auto p = _response->get_payload();
        reply_2_.set_data(std::string(reinterpret_cast<const char*>(p->get_data()), 0, p->get_length()));
        
        responder_2_.Finish(reply_2_, Status::OK, this);

        status_ = FINISH;
    }
}

void 
Adapter::HandleRpcs()
{
    new CallData(*this, &service_, cq_.get(), true);
    new CallData(*this, &service_, cq_.get(), false);
    
    void* tag;
    bool ok;

    while (!done_.load())
    {
        CHECK(cq_->Next(&tag, &ok));

        if(ok)
        {
            static_cast<CallData*>(tag)->Proceed();
        }

        CHECK(cq_->Next(&tag, &ok));

        if(ok)
        {
            static_cast<CallData*>(tag)->Proceed();
        }
    }
}

void
Adapter::on_state_cbk(vsomeip::state_type_e _state)
{
    if(_state == vsomeip::state_type_e::ST_REGISTERED)
    {
        app_->get_offered_services_async(vsomeip::offer_type_e::OT_ALL, std::bind(&Adapter::on_offered_services_all, this, std::placeholders::_1));
    }
}

void
Adapter::on_availability_cbk(vsomeip::service_t _service, vsomeip::instance_t _instance, bool _is_available)
{
    auto service = std::make_pair(_service, _instance);
    if(_is_available)
    {
        if(std::find(services_.begin(), services_.end(), service) == services_.end())
        {
            services_.push_back(std::move(service));

            app_->request_service(_service, _instance);
        }
    }
    else
    {
        auto found_service = std::remove(services_.begin(), services_.end(), service);
        if(found_service != services_.end())
        {
            services_.erase(found_service, services_.end());

            app_->release_service(_service, _instance);
        }
    }
}

void
Adapter::on_offered_services_all(const std::vector<std::pair<vsomeip::service_t, vsomeip::instance_t>> &_services)
{
    auto current(_services);
    std::sort(std::begin(current), std::end(current));

    std::size_t i = 0, j= 0;
    for (; i < services_.size() && j < current.size();) {
        if (services_[i] == current[j]) {
            i++;
            j++;
        } else if (services_[i] < current[j]) {
            app_->release_service(std::get<0>(services_[i]), std::get<1>(services_[i]));
            i++;
         } else {
            app_->request_service(std::get<0>(current[j]), std::get<1>(current[j]));
            j++;
         }
    }
    
    for(; i < services_.size(); ++i)
        app_->release_service(std::get<0>(services_[i]), std::get<1>(services_[i]));
    
    for(; j < current.size(); ++j)
        app_->request_service(std::get<0>(current[j]), std::get<1>(current[j]));

    services_ = current;
}