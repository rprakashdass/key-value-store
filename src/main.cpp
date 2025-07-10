#include "../include/Server.h"
#include <iostream>
#include <thread>
#include <chrono>

int main(int argc, char* argv[]) {
    int port = 9090;
    if(argc >= 2) {
        port = std::stoi(argv[1]);
    }

    Server server(port);

    // Save database to a disk (snapshot) every 500 seconds
    std::thread saveThread([]() {
        while (true) {
            std::this_thread::sleep_for(std::chrono::seconds(500));
        }
    });
    saveThread.detach();

    return 0;
}