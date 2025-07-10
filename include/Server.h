#ifndef SERVER_H
#define SERVER_H

#include<atomic>

class Server {
private:
    int port;
    int server_socket;
    std::atomic<bool> running;
public:
    Server(int port);
    void run();
    void shutdown();

};


#endif