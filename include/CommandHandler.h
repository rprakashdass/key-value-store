#ifndef COMMAND_HANDLER_H
#define COMMAND_HANDLER_H

#include<string>

class CommandHandler {
private:
public:
    CommandHandler();
    std::vector<std::string> parseCommand(const std::string& command);
    std::string processCommand(const std::string& commandLine);
};

#endif