#!/bin/bash

# List of servers (replace these with your actual server IPs or hostnames)
servers=(
    # "HOSTNAME_OR_IP_1"
    # "HOSTNAME_OR_IP_2"
    # "HOSTNAME_OR_IP_3"
    #  ...
)

# Name of the tmux session
ROOT_DIR="membership"
SESSION_NAME="session"
RUN_SERVER_CMD="cd $ROOT_DIR/server && go run ."

# Create a new tmux session in detached mode
tmux new-session -d -s $SESSION_NAME

# Loop through the servers and open a new window for each
for i in "${!servers[@]}"; do
    if [ "$i" -eq 0 ]; then

        # scp file over
        scp -r ~/$ROOT_DIR/* ${servers[$i]}:/home/${USER}/$ROOT_DIR/ 1> /dev/null 

        # run server
        tmux send-keys -t $SESSION_NAME "ssh ${servers[$i]} -o StrictHostKeyChecking=no" C-m
        tmux send-keys -t $SESSION_NAME "$RUN_SERVER_CMD" C-m
    else
        # scp file over
        scp -r ~/$ROOT_DIR/* ${servers[$i]}:/home/${USER}/$ROOT_DIR/ 1> /dev/null 

        # run server
        tmux new-window -t $SESSION_NAME "ssh ${servers[$i]} -o StrictHostKeyChecking=no"
        tmux send-keys -t $SESSION_NAME "$RUN_SERVER_CMD" C-m
    fi
done

wait

# Attach to the tmux session
tmux attach -t $SESSION_NAME
