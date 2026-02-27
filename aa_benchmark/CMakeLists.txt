cmake_minimum_required(VERSION 3.13)

project(AraCM_Method_Benchmark CXX)

find_package(com-communication-manager CONFIG REQUIRED)

set(SERVER_APP_NAME Publisher)
set(CLIENT_APP_NAME Subscriber)

if(EXISTS ${CMAKE_CURRENT_SOURCE_DIR}/arxml/AraCM_Method_Benchmark.arxml)
   execute_process(COMMAND ${VRTE_FS} -b ${CMAKE_CURRENT_SOURCE_DIR} -fp **.arxml comapi -a  -o ${CMAKE_CURRENT_SOURCE_DIR}/gen)
else()
    message(FATAL_ERROR "AraCM_Method_Benchmark.arxml Not found, check AraCM_Method_Benchmark configuration")
endif()

if(EXISTS ${CMAKE_CURRENT_SOURCE_DIR}/project_ServiceInstanceManifest.arxml)
   execute_process(COMMAND ${VRTE_FS} -b ${CMAKE_CURRENT_SOURCE_DIR} -fp **.arxml dsomeip -o ${CMAKE_CURRENT_SOURCE_DIR}/gen)
else()
    message(FATAL_ERROR "project_ServiceInstanceManifest.arxml Not found, check AraCM_Method_Benchmark configuration")
endif()

execute_process(COMMAND ${VRTE_FS} -b ${CMAKE_CURRENT_SOURCE_DIR} -fp **.arxml flatbuffers  -o ${CMAKE_CURRENT_SOURCE_DIR}/gen -flatc ${FLATC})

add_executable(${SERVER_APP_NAME} ${CMAKE_CURRENT_SOURCE_DIR}/publisher/publisher.cpp ${CMAKE_CURRENT_SOURCE_DIR}/gen/com/mercedes_benz/method_benchmark/aracm_method_benchmark_skeleton.cpp ${CMAKE_CURRENT_SOURCE_DIR}/gen/com/mercedes_benz/method_benchmark/aracm_method_benchmark_factory.cpp)
add_executable(${CLIENT_APP_NAME} ${CMAKE_CURRENT_SOURCE_DIR}/subscriber/subscriber.cpp ${CMAKE_CURRENT_SOURCE_DIR}/gen/com/mercedes_benz/method_benchmark/aracm_method_benchmark_proxy.cpp ${CMAKE_CURRENT_SOURCE_DIR}/gen/com/mercedes_benz/method_benchmark/aracm_method_benchmark_factory.cpp)

target_include_directories(${SERVER_APP_NAME} PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/gen)
target_include_directories(${CLIENT_APP_NAME} PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/gen)

target_link_libraries(${SERVER_APP_NAME} PRIVATE COM::com-communication-manager $<$<PLATFORM_ID:Linux>:rt> $<$<PLATFORM_ID:Linux>:pthread>)
target_link_libraries(${CLIENT_APP_NAME} PRIVATE COM::com-communication-manager $<$<PLATFORM_ID:Linux>:rt> $<$<PLATFORM_ID:Linux>:pthread>)

install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/softwarecluster_0/exm__SoftwareCluster_0_flatcfg.bin DESTINATION bin)
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/softwarecluster_1/exm__SoftwareCluster_1_flatcfg.bin DESTINATION bin)

install(FILES gen/com__SWCL_1_flatcfg.bin DESTINATION bin)

install(FILES gen/softwarecluster_0/comclient_publisher_process__SoftwareCluster_0_flatcfg.bin DESTINATION etc/ecu-cfg/processes/Publisher)
install(FILES gen/softwarecluster_1/comclient_subscriber_process__SoftwareCluster_1_flatcfg.bin DESTINATION etc/ecu-cfg/processes/Subscriber)

install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/softwarecluster_0/log_dlt_a__SoftwareCluster_0_flatcfg.bin DESTINATION bin)
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/softwarecluster_1/log_dlt_b__SoftwareCluster_1_flatcfg.bin DESTINATION bin)

install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/softwarecluster_0/log_publisher_process__SoftwareCluster_0_flatcfg.bin DESTINATION etc/ecu-cfg/processes/Publisher)
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/softwarecluster_1/log_subscriber_process__SoftwareCluster_1_flatcfg.bin DESTINATION etc/ecu-cfg/processes/Subscriber)
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/softwarecluster_0/log_comd_a__SoftwareCluster_0_flatcfg.bin DESTINATION bin)
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/softwarecluster_1/log_comd_b__SoftwareCluster_1_flatcfg.bin DESTINATION bin)
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/machine_a/dltd__Machine_A_flatcfg.bin DESTINATION bin)
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/softwarecluster_0/log_dltd__SoftwareCluster_0_flatcfg.bin DESTINATION bin)
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/machine_b/dltd__Machine_B_flatcfg.bin DESTINATION bin)
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/gen/softwarecluster_1/log_dltd__SoftwareCluster_1_flatcfg.bin DESTINATION bin)

install(FILES dlt_config/dlt_a.conf DESTINATION etc)
install(FILES dlt_config/dlt_b.conf DESTINATION etc)

install(TARGETS ${SERVER_APP_NAME} DESTINATION bin COMPONENT ${SERVER_APP_NAME}
    PERMISSIONS OWNER_READ OWNER_WRITE OWNER_EXECUTE)
install(TARGETS ${CLIENT_APP_NAME} DESTINATION bin COMPONENT ${CLIENT_APP_NAME}
     PERMISSIONS OWNER_READ OWNER_WRITE OWNER_EXECUTE)
     
# Install custom benchmarking script
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/scripts/collect-stats.sh DESTINATION bin PERMISSIONS OWNER_READ OWNER_WRITE OWNER_EXECUTE)     

