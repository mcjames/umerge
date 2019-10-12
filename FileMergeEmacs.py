# FileMergeEmacs.py

import os
import shutil
import tempfile


class FileMergeEmacs:

    def __init__( self ):
        pass

    def destroy( self ):
        pass

    def diff( self, left, right ):
        #print( "Left:", left )
        #print( "Right:", right )

        command = 'emacs -nw --eval "(ediff \\"'
        command += left
        command += '\\" \\"'
        command += right
        command += '\\")"'

        self.run_command( command )


    def make_temp_ancestor( self, source_pathname ):
        # fixme: error handling. Should spawn "cp", since on a Mac that
        # will copy Metadata properly. I'm not sure if emacs would handle
        # it, though, so that may be moot.
        tempname = tempfile.mktemp()
        #print( 'source:', source_pathname )
        #print( 'dest:', tempname )

        shutil.copy( source_pathname, tempname )

        return tempname


    def merge( self, left, middle, right ):
        #print( "Left:", left )
        #print( "Middle:", middle )
        #print( "Right:", right )

        # fixme: Need a signal handler to delete this?
        temp_ancestor = self.make_temp_ancestor( middle )

        # ediff-merge-files-with-ancestor wants ancestor as the last arg
        command = 'emacs -nw --eval "(progn (setq backup-inhibited t)(setq auto-save-default nil)(ediff-merge-files-with-ancestor \\"'
        #command = 'emacs -nw --eval "(progn (setq backup-inhibited t)(auto-save-mode -1)(ediff-merge-files-with-ancestor \\"'
        command += left
        command += '\\" \\"'
        command += right
        command += '\\" \\"'
        command += temp_ancestor
        command += '\\" nil \\"'
        command += middle
        command += '\\"))"'

        self.run_command( command )

        os.remove( temp_ancestor )


    def view( self, filename ):
        command = 'emacs -nw "'
        command += filename
        command += '"'

        self.run_command( command )


    def run_command( self, command ):
        # fixme: Modify this to handle errors.
        thepid = os.fork()
        if thepid == 0:
            # child
            os.execl( "/bin/sh", "sh", "-c", command )
        else:
            # parent
            os.waitpid( thepid, 0 )

