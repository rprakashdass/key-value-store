#include <iostream>
#include <sys/socket.h>
#include <netinet/in.h>
#include <unistd.h>
#include <string>
#include <thread>
#include <vector>
#include <queue>
#include <mutex>
#include <condition_variable>
#include <functional>
#include <sstream>
#include <unordered_map>

#define PORT 8080
#define THREAD_POOL_SIZE 4

// --- Thread Pool Implementation ---
class ThreadPool {
public:
    ThreadPool(size_t num_threads) : stop(false) {
        for (size_t i = 0; i < num_threads; ++i) {
            workers.emplace_back([this] {
                while (true) {
                    std::function<void()> task;
                    {
                        std::unique_lock<std::mutex> lock(queue_mutex);
                        condition.wait(lock, [this] { return stop || !tasks.empty(); });
                        if (stop && tasks.empty()) return;
                        task = std::move(tasks.front());
                        tasks.pop();
                    }
                    task();
                }
            });
        }
    }

    template<class F>
    void enqueue(F&& task) {
        {
            std::unique_lock<std::mutex> lock(queue_mutex);
            tasks.emplace(std::forward<F>(task));
        }
        condition.notify_one();
    }

    ~ThreadPool() {
        {
            std::unique_lock<std::mutex> lock(queue_mutex);
            stop = true;
        }
        condition.notify_all();
        for (std::thread &worker : workers) {
            worker.join();
        }
    }

private:
    std::vector<std::thread> workers;
    std::queue<std::function<void()>> tasks;
    std::mutex queue_mutex;
    std::condition_variable condition;
    bool stop;
};

// --- HTTP Request Parsing ---
struct HTTPRequest {
    std::string method;
    std::string path;
    std::unordered_map<std::string, std::string> headers;
};

HTTPRequest parse_request(const std::string& raw_request) {
    HTTPRequest req;
    std::istringstream iss(raw_request);
    std::string line;

    // Parse method and path (first line)
    std::getline(iss, line);
    std::istringstream first_line(line);
    first_line >> req.method >> req.path;

    // Parse headers
    while (std::getline(iss, line) && line != "\r") {
        size_t colon_pos = line.find(':');
        if (colon_pos != std::string::npos) {
            std::string key = line.substr(0, colon_pos);
            std::string value = line.substr(colon_pos + 2, line.size() - colon_pos - 3);
            req.headers[key] = value;
        }
    }

    return req;
}

// --- HTTP Response Builder ---
std::string build_response(const std::string& content, const std::string& content_type = "text/plain", int status_code = 200) {
    std::string status_text;
    switch (status_code) {
        case 200: status_text = "OK"; break;
        case 404: status_text = "Not Found"; break;
        default: status_text = "Unknown";
    }

    std::ostringstream oss;
    oss << "HTTP/1.1 " << status_code << " " << status_text << "\r\n"
        << "Content-Type: " << content_type << "\r\n"
        << "Content-Length: " << content.size() << "\r\n"
        << "Connection: close\r\n"
        << "\r\n"
        << content;

    return oss.str();
}

// --- Client Handler ---
void handle_client(int client_socket) {
    char buffer[4096] = {0};
    read(client_socket, buffer, 4096);

    HTTPRequest req = parse_request(buffer);
    std::string response;

    if (req.path == "/") {
        response = build_response("Welcome to the C++ HTTP Server!");
    } else if (req.path == "/data") {
        response = build_response("{\"message\":\"Hello JSON!\"}", "application/json");
    } else {
        response = build_response("404 Page Not Found", "text/html", 404);
    }

    send(client_socket, response.c_str(), response.size(), 0);
    close(client_socket);
}

// --- Main Server ---
int main() {
    int server_fd, client_socket;
    struct sockaddr_in address;
    int opt = 1;
    int addrlen = sizeof(address);

    // Create socket
    if ((server_fd = socket(AF_INET, SOCK_STREAM, 0)) < 0) {
        perror("socket failed");
        exit(EXIT_FAILURE);
    }

    // Set socket options
    if (setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR | SO_REUSEPORT, &opt, sizeof(opt))) {
        perror("setsockopt");
        exit(EXIT_FAILURE);
    }

    address.sin_family = AF_INET;
    address.sin_addr.s_addr = INADDR_ANY;
    address.sin_port = htons(PORT);

    // Bind and listen
    if (bind(server_fd, (struct sockaddr*)&address, sizeof(address)) < 0) {
        perror("bind failed");
        exit(EXIT_FAILURE);
    }

    if (listen(server_fd, 10) < 0) {
        perror("listen");
        exit(EXIT_FAILURE);
    }

    std::cout << "Server listening on port " << PORT << std::endl;

    // Thread pool
    ThreadPool pool(THREAD_POOL_SIZE);

    // Main accept loop
    while (true) {
        if ((client_socket = accept(server_fd, (struct sockaddr*)&address, (socklen_t*)&addrlen)) < 0) {
            perror("accept");
            continue;
        }

        // Handle client in thread pool
        pool.enqueue([client_socket]() {
            handle_client(client_socket);
        });
    }

    close(server_fd);
    return 0;
}