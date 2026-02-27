cmake_minimum_required (VERSION 3.22)
project (CARISMASomeIPAdapter)

find_package(utf8_range REQUIRED)
find_package(Protobuf REQUIRED)
find_package(gRPC CONFIG REQUIRED)
find_package(Threads REQUIRED)

set(VSOMEIP_NAME "vsomeip3")

set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -std=c++17")

include_directories(${PROTOBUF_INCLUDE_DIR})
include_directories(${GRPC_INCLUDE_DIRS})
include_directories(${VSOMEIP_INCLUDE_DIRS})

find_package(${VSOMEIP_NAME})
if (NOT ${VSOMEIP_NAME}_FOUND)
    message("${VSOMEIP_NAME} was not found. Please specify vsomeip_DIR")
endif()

add_executable (light_sensor_service light_sensor_service.cpp light_sensor_service_main.cpp)
target_link_libraries(light_sensor_service ${VSOMEIP_LIBRARIES} ${CMAKE_THREAD_LIBS_INIT})

add_executable (adapter adapter.grpc.pb.cc adapter.pb.cc adapter.cpp adapter_main.cpp)
target_link_libraries(adapter ${PROTOBUF_LIBRARY} utf8_range::utf8_validity gRPC::grpc++ ${VSOMEIP_LIBRARIES} ${CMAKE_THREAD_LIBS_INIT})
