// SPDX-License-Identifier: MIT
// Copyright (c) 2026 MBition GmbH

#ifndef INCLUDED_ADAPTER_H
#define INCLUDED_ADAPTER_H

// ---- STL ----
#include <atomic>
#include <cstdio>
#include <memory>
#include <string>
#include <thread>
#include <utility>
#include <vector>

// ---- VSOMEIP ----
#include <vsomeip/vsomeip.hpp>

// ---- gRPC----
#include "adapter.grpc.pb.h"

#include <absl/log/check.h>
#include <grpcpp/grpcpp.h>

#define LOG_INF(...) fprintf(stdout, __VA_ARGS__), fprintf(stdout, "\n")
#define LOG_ERR(...) fprintf(stderr, __VA_ARGS__), fprintf(stderr, "\n")

using grpc::Server;
using grpc::ServerAsyncResponseWriter;
using grpc::ServerBuilder;
using grpc::ServerCompletionQueue;
using grpc::ServerContext;
using grpc::Status;

using AdapterService = com::mercedes_benz::csa::Adapter;

using google::protobuf::Empty;

using com::mercedes_benz::csa::Method;
using com::mercedes_benz::csa::MethodCallResponse;
using com::mercedes_benz::csa::Services;
using com::mercedes_benz::csa::Service;

class Adapter {
public:
    Adapter()
        : rtm_(vsomeip::runtime::get())
        , app_(rtm_->create_application())
        , done_(false)
    {}
    bool init();
    void start();
    void stop();
private:
    // ---- gRPC----
    class CallData {
       public:
        CallData(Adapter& _adapter, AdapterService::AsyncService* _service, ServerCompletionQueue* _cq, bool _diff)
            : adapter_(_adapter)
            , service_(_service)
            , cq_(_cq)
            , diff_(_diff)
            , status_(CREATE)
            , responder_(&ctx_)
            , responder_2_(&ctx_)
            , session_(0)
        {
            Proceed();
        }
        void Proceed();
    private:
        // ---- VSOMEIP ----
        void on_message_cbk(const std::shared_ptr<vsomeip::message> &_response);
        
        // ---- gRPC ----
        Adapter& adapter_;
        
        AdapterService::AsyncService* service_;
        ServerCompletionQueue* cq_;
        ServerContext ctx_;

        // RPC 1 / RPC 2?
        bool diff_;

        enum CallStatus { CREATE, PROCESS, FINISH };
        CallStatus status_;

        // ---- RPC 1: GET SERVICES ----
        Empty request_;
        Services reply_;
        ServerAsyncResponseWriter<Services> responder_;

        // ---- RPC 2: CALL METHOD----
        Method request_2_;
        MethodCallResponse reply_2_;
        ServerAsyncResponseWriter<MethodCallResponse> responder_2_;

        vsomeip::service_t _sid;
        vsomeip::instance_t _iid;
        vsomeip::method_t _mid;
        vsomeip::session_t session_;
    };

    void HandleRpcs();

    std::atomic_bool done_ {};
    std::unique_ptr<std::thread> worker_;

    std::unique_ptr<ServerCompletionQueue> cq_;
    AdapterService::AsyncService service_;
    std::unique_ptr<Server> server_;

    // ---- VSOMEIP ----
    void on_state_cbk(vsomeip::state_type_e _state);
    void on_availability_cbk(vsomeip::service_t _service, vsomeip::instance_t _instance, bool _is_available);
    void on_offered_services_all(const std::vector<std::pair<vsomeip::service_t, vsomeip::instance_t>> &_services);

    std::vector<std::pair<vsomeip::service_t, vsomeip::instance_t>> services_;

    std::shared_ptr<vsomeip::runtime> rtm_;
    std::shared_ptr<vsomeip::application> app_;
};

#endif // INCLUDED_ADAPTER_H