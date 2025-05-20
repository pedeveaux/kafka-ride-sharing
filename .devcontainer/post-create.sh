#!/bin/bash

# Install plugins
git clone https://github.com/zsh-users/zsh-autosuggestions ~/.oh-my-zsh/custom/plugins/zsh-autosuggestions
git clone https://github.com/zsh-users/zsh-syntax-highlighting.git ~/.oh-my-zsh/custom/plugins/zsh-syntax-highlighting
git clone https://github.com/djui/alias-tips.git ~/.oh-my-zsh/custom/plugins/alias-tips
git clone https://github.com/zsh-users/zsh-completions.git ~/.oh-my-zsh/custom/plugins/zsh-completions

# Update .zshrc file
sed -i 's/^plugins=.*/plugins=(git zsh-autosuggestions zsh-syntax-highlighting alias-tips zsh-completions)/' ~/.zshrc

# Set the theme to agnoster
sed -i 's/ZSH_THEME=.*/ZSH_THEME="agnoster"/' ~/.zshrc


echo 'export GOPATH=$HOME' >> ~/.zshrc
echo 'export PATH=$PATH:$GOROOT/bin:$GOPATH/bin' >> ~/.zshrc
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.zshrc

# go install github.com/brianvoe/gofakeit/v6@latest
# go install github.com/confluentinc/confluent-kafka-go/kafka@latest

# Reload .zshrc file - This isn't needed as this is only run when the container is created
# source ~/.zshrc
