#include "../include/Server.h"

#include <iostream>
#include <vector>
#include <thread>
#include <sys/socket.h>
#include <sys/unistd.h>
#include <netinet/in.h>


static Server *globalServer = nullptr;

Server::Server(int port) : port(port) {
    server_socket = -1; // not allocated
    running = true;
    globalServer = this;
};


void Server::run() {
    server_socket = socket(AF_INET, SOCK_STREAM, 0);
    if(server_socket == -1) {
        std::cerr << "Could'nt create server socket!\n";
        return;
    }

    sockaddr_in serverAddr;
    serverAddr.sin_family = AF_INET;
    serverAddr.sin_port = htons(port);
    serverAddr.sin_addr.s_addr = INADDR_ANY;

    if(bind(server_socket, (sockaddr*)&serverAddr, sizeof(serverAddr)) == -1) {
        std::cerr << "Error on Binding Server Socket\n";
        return;
    }

    if(listen(server_socket, 3) == -1) {
        std::cerr << "Error Listening on Server\n";
        return;
    }

    std::cout << "DB Server Listening on port " << port << std::endl;


    std::vector<std::thread> clientThreads;
    while(running) {
        int client_socket = accept(server_socket, nullptr, nullptr);
        if(client_socket == -1) {
            std::cerr << "[Error] Can't accept the client\n";
            break;;
        }

    }

}


void Server::shutdown() {
    running = false;
    if(server_socket != -1) {
        close(server_socket);
    }
    std::cout << "Server stopped running\n";
}
