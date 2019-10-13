# FileMergeVim.py

import os


class FileMergeVim:

    def __init__(self):
        pass

    def destroy(self):
        pass

    def diff(self, left, right):
        # print( "Left:", left )
        # print( "Right:", right )

        command = 'vimdiff "'
        command += left
        command += '" "'
        command += right
        command += '"'

        self.run_command(command)

    def view(self, filename):
        command = 'vim "'
        command += filename
        command += '"'

        self.run_command(command)

    def run_command(self, command):
        # fixme: Modify this to handle errors.
        thepid = os.fork()
        if thepid == 0:
            # child
            os.execl("/bin/sh", "sh", "-c", command)
        else:
            # parent
            os.waitpid(thepid, 0)
