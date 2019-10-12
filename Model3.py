# Model3.py

import Match3
import os
#import shutil
#import tempfile
import threading
import time


NORMAL   = 1
MISMATCH = 2
CONFLICT = 3

# Current state of the model
STATE_NORMAL       = 0
STATE_NEW          = 1
STATE_ENUMERATING  = 2
STATE_PRECOMPARING = 3
STATE_COMPARING    = 4
STATE_COPYING      = 5
STATE_DELETING     = 6
STATE_MERGING      = 7


################################################################
#
#
class Model:

    ################################################################
    #
    #
    def __init__( self, fileops, left, middle, right ):
        self.fileops = fileops
        self.left = left
        self.middle = middle
        self.right = right
        self.modelstate = STATE_NEW
        self.render_again = False

        self.tree_structure_lock = threading.Lock()

        self.operation_thread = None
        self.operation_arg = None

        # For copying only
        self.operation_copy_src = None
        self.operation_copy_dest = None

        self.comparison_thread = None
        self.stop_comparison = False

        self.top = Match3.Match( left, middle, right, None )


    ################################################################
    #
    #
    def destroy( self ):
        # fixme: Kill any threads here.
        pass


    ################################################################
    #
    #
    def state( self ):
        return self.modelstate


    ################################################################
    #
    #
    def selection_exists( self ):
        return self.top.selection_exists()



    def start_enumerate( self ):
        if self.__validate_item( self.top ):
            self.__initiate_enumerate()
            return True
        else:
            # Fixme: what should we do here? Or should the validation
            # already be done?
            return False

    #
    # Operation thread
    #

    ################################################################
    #
    #
    def request_operation( self, operation, item=None ):

        # An operation is currently in progress, so signal an error
        # and do nothing.
        if self.operation_thread:
            return False

        # If this operation was requested for an item that has
        # descendants that are still uncompared, signal an error and
        # do nothing.
        if operation != "refresh" and item is not None and item.has_uncompared_descendants():
            #print( "has uncompared descendants. Doing nothing..." )
            return False

        if operation == "refresh":
            if item is not None:
                item.set_state_of_tree( Match3.UNCOMPARED )
            self.__initiate_compare()
        elif operation == "copy_a2b":
            # Fixme: do this in a smarter way
            self.operation_arg = item
            self.operation_copy_src = 'a'
            self.operation_copy_dest = 'b'
            self.__initiate_copy()
        elif operation == "copy_a2c":
            self.operation_arg = item
            self.operation_copy_src = 'a'
            self.operation_copy_dest = 'c'
            self.__initiate_copy()
        elif operation == "copy_b2a":
            self.operation_arg = item
            self.operation_copy_src = 'b'
            self.operation_copy_dest = 'a'
            self.__initiate_copy()
        elif operation == "copy_b2c":
            self.operation_arg = item
            self.operation_copy_src = 'b'
            self.operation_copy_dest = 'c'
            self.__initiate_copy()
        elif operation == "copy_c2a":
            self.operation_arg = item
            self.operation_copy_src = 'c'
            self.operation_copy_dest = 'a'
            self.__initiate_copy()
        elif operation == "copy_c2b":
            self.operation_arg = item
            self.operation_copy_src = 'c'
            self.operation_copy_dest = 'b'
            self.__initiate_copy()
        elif operation == 'merge_to_center':
            self.__initiate_merge_to_center( item )
        elif operation == "copy_l2r":
            self.__initiate_copy_l2r( item )
        elif operation == "copy_r2l":
            self.__initiate_copy_r2l( item )
        elif operation == "delete":
            return self.__initiate_delete( item )
        else:
            pass # FIXME: flag an error here.


    ################################################################
    #
    #
    def __validate_item( self, item ):
        #print( "in __validate_item()" )
        return ((item.left is not None and os.path.exists( item.left ))
                or (item.middle is not None and os.path.exists( item.middle ))
                or (item.right is not None and os.path.exists( item.right )))


    ################################################################
    #
    #
    def __initiate_enumerate( self ):
        #print( "in __initiate_enumerate()" )
        self.modelstate = STATE_ENUMERATING
        self.operation_thread = threading.Thread( target=self.__enumerate_aux )
        #self.operation_arg = item
        self.operation_thread.start()


    ################################################################
    #
    #
    def __enumerate_aux( self ):
        with self.lock():
            #print( 'start of enumerate lock' )
            self.top.enumerate()
            self.top.set_state_of_tree( Match3.UNCOMPARED )
            self.top.lm_num_diffs = None
            self.top.mr_num_diffs = None
            self.top.lr_num_diffs = None
            if self.top.resolution_status != ' ':
                self.top.set_resolution_status_of_tree( self.top.resolution_status )
            #print( 'end of enumerate lock' )
        self.modelstate = STATE_PRECOMPARING
        self.operation_thread = None # Is this the right place to do this?
        #self.operation_arg = None

        try:
            self.stop_comparison = True
            #print( "Stopping thread..." )
            #print( "thread=", self.comparison_thread )
            #print( "is_alive()=", self.comparison_thread.is_alive() )
            if self.comparison_thread:
                self.comparison_thread.join( 1.0 )
            #print( "thread=", self.comparison_thread )
            #print( "is_alive()=", self.comparison_thread.is_alive() )
        except:
            pass

        self.__initiate_compare()


    #
    # Comparison
    #

    ################################################################
    #
    #
    def __initiate_compare( self ):
        self.modelstate = STATE_COMPARING
        self.stop_comparison = False
        #print( "Creating comparison thread..." )
        self.comparison_thread = threading.Thread( target=self.__compare_aux )
        #print( "new thread:", self.comparison_thread )
        self.comparison_thread.start()


    ################################################################
    #
    #
    def __compare_aux( self ):
        self.compare_match_item( self.top )
        self.render_again = True
        self.modelstate = STATE_NORMAL
        #self.comparison_thread = None # Is this the right place to do this?


    ################################################################
    #
    #
    def compare_match_item( self, item ):
        if self.stop_comparison:
            return
        if item.state == Match3.UNCOMPARED:
            item.compare_myself( self.fileops )
        for child in item.children:
            self.compare_match_item( child )


    ################################################################
    #
    #
    def lock( self ):
        return self.tree_structure_lock


    #
    # Operations
    #

    ################################################################
    #
    #
    def __initiate_copy( self ):
        #print( "in __initiate_copy()" )

        #print( "src :", self.operation_copy_src )
        #print( "dest:", self.operation_copy_dest )

        self.modelstate = STATE_COPYING
        self.operation_thread = threading.Thread( target=self.__copy_aux )
        self.operation_thread.start()


    ################################################################
    #
    #
    def __copy_aux( self ):
        item = self.operation_arg

#         if item.left is None:
#             return

#         left_name = item.left_pathname()
#         right_name = item.right_pathname()

        src = item.letter_to_subpart( self.operation_copy_src )
        dest = item.letter_to_subpart( self.operation_copy_dest )

#         src = self.operation_copy_src
#         dest = self.operation_copy_dest

        if os.path.exists( dest ) and os.path.isdir( dest ):
            result = self.fileops.delete_primitive( dest )
            if result is not None:
                # fixme: do something with an error
                pass

        # Probably need to catch any exceptions here
        # Also, we need a loop here watching a variable to know if we
        # have been asked to cancel the operation.  Mark it as error
        # and then just return cleanly.  User terminate() or kill()
        # method on p1.
        result = self.fileops.copy_primitive( src, dest )

        if result is None:
            # For now we can just start a refresh on the top item.  It
            # might be more efficient to just alter the items, eventually.
            if self.operation_copy_dest == 'a':
                item.left = dest
            elif self.operation_copy_dest == 'b':
                item.right = dest
            elif self.operation_copy_dest == 'c':
                item.middle = dest

            self.operation_thread = None
            self.request_operation( "refresh", item )
        else:
            item.set_state_of_tree( Match3.ERROR )
            self.modelstate = STATE_NORMAL
            self.operation_thread = None


    ################################################################
    #
    #
    def __initiate_copy_l2r( self, item ):
        #print( "in __initiate_copy_l2r()" )

        #print( "left :", item.left_pathname() )
        #print( "right:", item.right_pathname() )

        self.modelstate = STATE_COPYING
        self.operation_thread = threading.Thread( target=self.__copy_l2r_aux )
        self.operation_arg = item
        self.operation_thread.start()


    ################################################################
    #
    #
    def __copy_l2r_aux( self ):
        item = self.operation_arg

        if item.left is None:
            return

        left_name = item.left_pathname()
        right_name = item.right_pathname()

        if os.path.exists( right_name ) and os.path.isdir( right_name ):
            result = self.fileops.delete_primitive( right_name )
            if result is not None:
                # fixme: do something with an error
                pass

        # Probably need to catch any exceptions here
        # Also, we need a loop here watching a variable to know if we
        # have been asked to cancel the operation.  Mark it as error
        # and then just return cleanly.  User terminate() or kill()
        # method on p1.
        result = self.fileops.copy_primitive( left_name, right_name )

        if result is None:
            # For now we can just start a refresh on the top item.  It
            # might be more efficient to just alter the items, eventually.
            item.right = right_name
            self.operation_thread = None
            self.request_operation( "refresh", item )
        else:
            item.set_state_of_tree( Match3.ERROR )
            self.modelstate = STATE_NORMAL
            self.operation_thread = None


    ################################################################
    #
    #
    def __initiate_copy_r2l( self, item ):
        #print( "in __initiate_copy_r2l()" )

        #print( "left :", item.left_pathname() )
        #print( "right:", item.right_pathname() )

        self.modelstate = STATE_COPYING
        self.operation_thread = threading.Thread( target=self.__copy_r2l_aux )
        self.operation_arg = item
        self.operation_thread.start()


    ################################################################
    #
    #
    def __copy_r2l_aux( self ):
        item = self.operation_arg

        if item.right is None:
            return

        left_name = item.left_pathname()
        right_name = item.right_pathname()

        if os.path.exists( left_name ) and os.path.isdir( left_name ):
            result = self.fileops.delete_primitive( left_name )
            if result is not None:
                # fixme: do something with an error
                pass

        # Probably need to catch any exceptions here
        # Also, we need a loop here watching a variable to know if we
        # have been asked to cancel the operation.  Mark it as error
        # and then just return cleanly.  User terminate() or kill()
        # method on p1.
        result = self.fileops.copy_primitive( right_name, left_name )

        if result is None:
            # For now we can just start a refresh on the top item.  It
            # might be more efficient to just alter the items, eventually.
            item.left = left_name
            self.operation_thread = None
            self.request_operation( "refresh", item )
        else:
            item.set_state_of_tree( Match3.ERROR )
            self.modelstate = STATE_NORMAL
            self.operation_thread = None


    ################################################################
    #
    #
    def __delete_item( self, item ):
        if item.left is not None:
            #print( "Deleting: ", item.left )
            pass
        elif item.middle is not None:
            #print( "Deleting: ", item.middle )
            pass
        else:
            #print( "Deleting: ", item.right )
            pass

        exit_status_left = None
        exit_status_middle = None
        exit_status_right = None

        if item.left is not None:
            exit_status_left = self.fileops.delete_primitive( item.left )
        if item.middle is not None:
            exit_status_middle = self.fileops.delete_primitive( item.middle )
        if item.right is not None:
            exit_status_right = self.fileops.delete_primitive( item.right )

        # If either failed, we need to mark the item red.
        if ( exit_status_left is not None
            or exit_status_middle is not None
            or exit_status_right is not None ):
            # fixme: Set error member of the item to error message
            item.set_state_of_tree( Match3.ERROR )
            #print( 'Returning False' )
            return False # don't need to reset cursor
        else:
            item.set_state_of_tree( Match3.DELETED )
            #print( 'Returning True' )
            return True # need to reset cursor


    ################################################################
    #
    #
    def __initiate_delete( self, item ):
        if item is not None:
            return self.__delete_item( item )
        else:
            # Walk the tree and delete any selected items.
            return self.__delete_selected_aux( self.top )


    ################################################################
    #
    #
    def __delete_selected_aux( self, item ):
        if item.selected:
            return_value = self.__delete_item( item )
        for child in item.children:
            if child.selected:
                self.__delete_item( child )
            else:
                self.__delete_selected_aux( child )
        return return_value


    ################################################################
    #
    #
    def __initiate_merge_to_center( self, item ):
        #print( "in __initiate_merge_to_center()" )

        self.modelstate = STATE_MERGING
        self.operation_thread = threading.Thread(
                                    target=self.__merge_to_center_thread )
        self.operation_arg = item
        self.operation_thread.start()


    ################################################################
    #
    #
    def __merge_to_center_thread( self ):
        item = self.operation_arg

        if item is not None:
            self.__merge_individual_item( item )
        else:
            # Walk the tree and merge any selected items.
            self.__merge_selected_aux( self.top )
        self.modelstate = STATE_NORMAL
        self.operation_thread = None
        self.request_operation( "refresh", item )


    ################################################################
    #
    #
    # Returns True only if all parts (l, m, r) present are files.
    def __all_are_files( self, item ):
        if item.left is not None and os.path.isdir( item.left ):
                return False
        if item.middle is not None and os.path.isdir( item.middle ):
                return False
        if item.right is not None and os.path.isdir( item.right ):
                return False
        return True


    ################################################################
    #
    #
    def __all_are_dirs( self, item ):
        if item.left is not None and not os.path.isdir( item.left ):
                return False
        if item.middle is not None and not os.path.isdir( item.middle ):
                return False
        if item.right is not None and not os.path.isdir( item.right ):
                return False
        return True


    ################################################################
    #
    #
    def __merge_individual_item( self, item ):
        if self.__all_are_files( item ):
            self.__merge_file_item( item )
        elif self.__all_are_dirs( item ):
            self.__merge_dir_item( item )
        else:
            item.resolution_status = 'c'


    ################################################################
    #
    #
    def __merge_dir_item( self, item ):
        #print( 'dir item:', item )

        if ( item.left is None
             and item.middle is not None
             and item.right is None ):
            #print( '  deleting:', item.middle )
            result = self.__delete_item( item.middle )
            if not result:
                # fixme: do something with an error
                #print( 'Error deleting', result )
                pass
            return

        if ( item.left is not None
             #and item.middle is None
             and item.right is None ):
            result = self.fileops.delete_primitive( item.middle )
            #print( '  copying:', item.left, 'to middle' )
            result = self.fileops.copy_primitive( item.left,
                                                  item.middle_pathname() )
            if result is not None:
                # fixme: do something with an error
                #print( 'Error copying', result )
                pass
            else:
                item.middle = item.middle_pathname()
                item.set_resolution_status_of_tree( 'a' )
            return

        if ( item.left is None
             #and item.middle is None
             and item.right is not None ):
            result = self.fileops.delete_primitive( item.middle )
            #print( '  copying:', item.right, 'to middle' )
            result = self.fileops.copy_primitive( item.right,
                                                  item.middle_pathname() )
            if result is not None:
                # fixme: do something with an error
                #print( 'Error copying', result )
                pass
            else:
                item.middle = item.middle_pathname()
                item.set_resolution_status_of_tree( 'b' )
            return

        if item.count() == 2:
            item.set_resolution_status_of_tree( 'c' )
        else:
            #print( '  recursing...' )
            for child in item.children:
                self.__merge_individual_item( child )


    ################################################################
    #
    #
    def __merge_file_item( self, item ):
        #print( 'file item:', item )

        if ( item.left is None
             and item.middle is not None
             and item.right is None ):
            #print( '  deleting:', item.middle )
            result = self.__delete_item( item )
            if not result:
                # fixme: do something with an error
                #print( 'Error deleting', result )
                pass
            return

        if ( item.left is not None
             #and item.middle is None
             and item.right is None ):
            #print( '  copying:', item.left, 'to middle' )
            result = self.fileops.copy_primitive( item.left,
                                                  item.middle_pathname() )
            if result is not None:
                # fixme: do something with an error
                #print( 'Error copying', result )
                pass
            else:
                item.middle = item.middle_pathname()
                item.set_resolution_status_of_tree( 'a' )
            return

        if ( item.left is None
             #and item.middle is None
             and item.right is not None ):
            #print( '  copying:', item.right, 'to middle' )
            result = self.fileops.copy_primitive( item.right,
                                                  item.middle_pathname() )
            if result is not None:
                # fixme: do something with an error
                #print( 'Error copying', result )
                pass
            else:
                item.middle = item.middle_pathname()
                item.set_resolution_status_of_tree( 'b' )
            return

        if ( item.left is not None
             and item.middle is None
             and item.right is not None
             and item.lr_num_diffs == 0 ):
            #print( '  copying:', item.right, 'to middle' )
            result = self.fileops.copy_primitive( item.right,
                                                  item.middle_pathname() )
            if result is not None:
                # fixme: do something with an error
                #print( 'Error copying', result )
                pass
            else:
                item.middle = item.middle_pathname()
                item.set_resolution_status_of_tree( 'b' )
            return

        if ( item.left is not None
             and item.middle is not None
             and item.right is not None ):
            self.__merge_file_all_three_present( item )
            return

        # If we made it to here, there is a merge conflict that the user
        # needs to address
        item.resolution_status = 'c'


    ################################################################
    #
    #
    def __merge_file_all_three_present( self, item ):
        #print( '  in __merge_file_all_three_present().' )

        if ( item.lm_num_diffs == 0
             and item.mr_num_diffs == 0
             and item.lr_num_diffs == 0 ):
            #print( '  no changes. Doing nothing.' )
            return

        # Test for merge conflicts here. If conflicted, mark it. Otherwise,
        # merge using diff3 and ed.
        if self.fileops.merge_conflicts_exist( item ):
            item.resolution_status = 'c'
        else:
            self.fileops.merge_files_to_base( item )
            item.set_resolution_status_of_tree( 'm' )


    ################################################################
    #
    #
    def __merge_selected_aux( self, item ):
        if item.selected:
            self.__merge_item( item )
        for child in item.children:
            if child.selected:
                self.__merge_item( child )
            else:
                self.__merge_selected_aux( child )


    ################################################################
    #
    #
    def merge_all( self ):
        self.__initiate_merge_to_center( self.top )
