# FileOpsPOSIX.py

import subprocess
import sys
import tempfile
import shutil


class FileOpsPOSIX:

    ################################################################
    #
    #
    def __init__(self):
        # Check for availability of the ops here?
        pass

    ################################################################
    #
    # Return None upon success, an error string upon failure.
    #
    def copy_primitive(self, src_pathname, dest_pathname):
        try:
            exit_status = 0
            if src_pathname is not None:
                # print( "src_pathname:", src_pathname )
                # print( "dest_pathname:", dest_pathname )
                p1 = subprocess.Popen(["cp", "-R", src_pathname,
                                      dest_pathname],
                                      stderr=sys.stdout)
                p1.wait()
                exit_status = p1.returncode
            if exit_status == 0:
                # print( "exit_status is zero" )
                return None
            else:
                # Fixme: how to localize this string?
                # print( "exit_status is nonzero", str(exit_status) )
                return 'cp: Non-zero return value: ' + str(exit_status)
        except Exception as e:
            return e

    ################################################################
    #
    # Return None upon success, an error string upon failure.
    #
    def delete_primitive(self, pathname):
        try:
            exit_status = 0
            if pathname is not None:
                p1 = subprocess.Popen(["rm", "-Rf", pathname],
                                      stderr=sys.stdout)
                p1.wait()
                exit_status = p1.returncode
            if exit_status == 0:
                return None
            else:
                # Fixme: how to localize this string?
                return 'rm: Non-zero return value: ' + str(exit_status)
        except Exception as e:
            return e

    ################################################################
    #
    #
    def merge_files_to_base(self, item):
        # print( "in merge_files_to_base()" )

        # fixme: tons of OS operations here. Need error handling.
        tempname = tempfile.mktemp()
        shutil.copy(item.middle, tempname)

        # print( '  left:', item.left )
        # print( 'middle:', item.middle )
        # print( ' right:', item.right )

        fd = open(item.middle, 'w')
        p1 = subprocess.Popen(['diff3', '-m', item.left, tempname,
                              item.right],
                              stdout=fd)
        p1.wait()
        fd.close()

        if p1.returncode == 0:
            # print( '--succeeded' )
            pass
        else:
            # fixme: handle an error here.
            # print( '...diff3 failed', p1.returncode )
            pass

    ################################################################
    #
    #
    def merge_conflicts_exist(self, item):
        # print( "in merge_conflicts_exist()" )
        p1 = subprocess.Popen(['diff3', '-x', item.left, item.middle,
                              item.right],
                              stdout=subprocess.PIPE,
                              stderr=sys.stdout)
        p1.wait()

        if p1.returncode == 0:
            output = p1.communicate()[0]
            # print( 'length:', len(output) )
            # print( 'output:', output, '\n' )
            return len(output) != 0
        else:
            # fixme: handle an error here.
            # print( '...diff3 failed' )
            pass

    ################################################################
    #
    #
    def compare_two_files(self, left, right):
        # print( 'comparing left :', left )
        # print( 'comparing right:', right )

        p1 = subprocess.Popen(["diff", left, right],
                              stdout=subprocess.PIPE)
        p2 = subprocess.Popen(["grep", "^[0-9]"],
                              stdin=p1.stdout, stdout=subprocess.PIPE)
        p3 = subprocess.Popen(["wc", "-l"], stdin=p2.stdout,
                              stdout=subprocess.PIPE)
        p1.wait()
        p2.wait()
        p3.wait()

        num_diffs = None
        if p3.returncode == 0:
            output = p3.communicate()[0]
            # print( 'output:', output )
            num_diffs = int(output)
        else:
            # print( 'the popen thing was terminated' )
            pass

        # print( '    num_diffs:', num_diffs )
        return num_diffs

#     def __compare_two_files( self, left, right ):
#         print( 'comparing left :', left )
#         print( 'comparing right:', right )

#         if self.left == None or self.right == None:
#             self.state = MISSING
#         elif os.path.isdir(self.left) and os.path.isdir(self.right):
#             self.state = SAME
#         elif os.path.isdir(self.left) != os.path.isdir(self.right):
#             #self.state = DIFFERENT
#             self.set_state_of_tree( DIFFERENT )
#         else:
#             command1 = "cmp -s " + self.left + " " + self.right
#             # If the sizes differ, we avoid a cmp.
#             if (os.path.getsize(self.left) == os.path.getsize(self.right)):
#                 #and os.system( command1 ) == 0):
#                 self.state = SAME
#             else:
#                 p1 = subprocess.Popen( ["diff", self.left, self.right],
#                                        stdout=subprocess.PIPE )
#                 p2 = subprocess.Popen( ["grep", "^[0-9]"],
#                                        stdin=p1.stdout,stdout=subprocess.PIPE)
#                 p3 = subprocess.Popen( ["wc", "-l"], stdin=p2.stdout,
#                                        stdout=subprocess.PIPE )
#                 p1.wait()
#                 p2.wait()
#                 p3.wait()

#                 if p3.returncode == 0:
#                     output = p3.communicate()[0]
#                     self.num_diffs = int(output)
#                     if self.num_diffs == 0: # need this?
#                         self.state = SAME
#                     else:
#                         self.state = DIFFERENT
#                 else:
#                     print( 'the popen thing was terminated' )

# Might be want some sort of optimization like this? modtimes are not very
# useful, but
#             if os.path.getsize(self.left) == os.path.getsize(self.right):
#                 if os.path.getmtime(self.left)==
#                   os.path.getmtime(self.right)):
#                     self.state = SAME

    ################################################################
    #
    #
    def compare_three_files(self, left, middle, right):
        # print()
        # print( "lmr: ", left, middle, right )
        p1 = subprocess.Popen(["diff3", left, middle, right],
                              stdout=subprocess.PIPE,
                              stderr=sys.stdout)
#                                   env={'PATH': '/usr/bin'})
        # fixme: is this regex right? does the star only apply to the
        # digit part?
        p2 = subprocess.Popen(["grep", "^====[[:digit:]]*"],
                              stdin=p1.stdout, stdout=subprocess.PIPE)
        p3 = subprocess.Popen(["sort"],
                              stdin=p2.stdout, stdout=subprocess.PIPE)
        p4 = subprocess.Popen(["uniq", "-c"],
                              stdin=p3.stdout, stdout=subprocess.PIPE)
        p1.wait()
        p2.wait()
        p3.wait()
        p4.wait()

        # num_diffs = None
        if p4.returncode == 0:
            output = p4.communicate()[0]
            # print( 'output:', output, '\n' )
            # num_diffs = int(output)
            lines = output.splitlines()
            lines = [str(x, 'utf-8') for x in lines]
            # print( 'lines:', lines )

            lm_count = 0
            mr_count = 0
            lr_count = 0

            for line in lines:
                if '====1' in line:
                    # print( 'found 1' )
                    num = int(line[:7])
                    lm_count += num
                    lr_count += num
                elif '====2' in line:
                    # print( 'found 2' )
                    num = int(line[:7])
                    lm_count += num
                    mr_count += num
                elif '====3' in line:
                    # print( 'found 3' )
                    num = int(line[:7])
                    mr_count += num
                    lr_count += num
                elif '====' in line:
                    # print( 'found all' )
                    num = int(line[:7])
                    lm_count += num
                    mr_count += num
                    lr_count += num

            # print( 'lm_count:', lm_count )
            # print( 'mr_count:', mr_count )
            # print( 'lr_count:', lr_count )

            return (lm_count, mr_count, lr_count)
        else:
            # print( 'the popen thing was terminated' )
            pass

        # print( '    num_diffs:', num_diffs )
        # return num_diffs

        return (None, None, None)
