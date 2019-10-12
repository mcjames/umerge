# Controller.py

import curses
import signal
import os
import Match


class Controller:

    def __init__( self, model, view, canvas, filemerge ):
        self.model = model
        self.view = view
        self.canvas = canvas
        self.filemerge = filemerge
        self.need_to_quit = False

        self.install_sigint_handler()

        if model.top.way() == 2:
            self.input_handler = self.two_way_normal_input
        else:
            self.input_handler = self.three_way_normal_input


    def destroy( self ):
        self.remove_sigint_handler()


    def install_sigint_handler( self ):
        signal.signal( signal.SIGINT, self.sigint_handler )


    def remove_sigint_handler( self ):
        signal.signal( signal.SIGINT, signal.SIG_DFL )


    def sigint_handler( self, signo, frame ):
        self.need_to_quit = True


    def process_input( self, key_value ):
        self.input_handler( key_value )


    def selection_type_input( self, key_value ):
        col = chr(self.chosen_column).upper()
        if key_value == ord('a'):
            #print( 'Selecting absent files from column ' + col )
            self.view.prompt = 'Column ' + col + ' absent files selected'
            self.model.top.select( col, 'a' )
        elif key_value == ord('u'):
            #print( 'Selecting unchanged files from column ' + col )
            self.view.prompt = 'Column ' + col + ' unchanged files selected'
        elif key_value == ord('c'):
            #print( 'Selecting changed files from column ' + col )
            self.view.prompt = 'Column ' + col + ' changed files selected'
        elif key_value == ord('i'):
            #print( 'Selecting inserted files from column ' + col )
            self.view.prompt = 'Column ' + col + ' inserted files selected'
        else:
            self.view.prompt = 'Invalid choice'

        self.chosen_column = None
        self.input_handler = self.three_way_normal_input


    def select_column_input( self, key_value ):
        if ( key_value == ord('a')
             or key_value == ord('b')
             or key_value == ord('c') ):
            self.chosen_column = key_value
            self.view.prompt = chr(key_value).upper() + ' -- which items:'
            self.input_handler = self.selection_type_input
        elif key_value == ord('x'):
            self.model.top.unselect_all()
            self.view.prompt = ''
            self.input_handler = self.three_way_normal_input
        else:
            self.view.prompt = 'Invalid choice'
            self.input_handler = self.three_way_normal_input



    def copy_a_3way_input( self, key_value ):
        if key_value == ord('b'):
            #print( 'Copying from A to B' )
            self.view.prompt = 'Copying from A to B'
            item = self.view.current_item()
            self.model.request_operation( "copy_a2b", item )
        elif key_value == ord('c'):
            #print( 'Copying from A to C' )
            self.view.prompt = 'Copying from A to C'
            item = self.view.current_item()
            self.model.request_operation( "copy_a2c", item )
        else:
            self.view.prompt = 'Invalid choice'

        self.input_handler = self.three_way_normal_input


    def copy_b_3way_input( self, key_value ):
        if key_value == ord('a'):
            #print( 'Copying from B to A' )
            self.view.prompt = 'Copying from B to A'
            item = self.view.current_item()
            self.model.request_operation( "copy_b2a", item )
        elif key_value == ord('c'):
            #print( 'Copying from B to C' )
            self.view.prompt = 'Copying from B to C'
            item = self.view.current_item()
            self.model.request_operation( "copy_b2c", item )
        else:
            self.view.prompt = 'Invalid choice'

        self.input_handler = self.three_way_normal_input


    def copy_c_3way_input( self, key_value ):
        if key_value == ord('a'):
            #print( 'Copying from C to A' )
            self.view.prompt = 'Copying from C to A'
            item = self.view.current_item()
            self.model.request_operation( "copy_c2a", item )
        elif key_value == ord('b'):
            #print( 'Copying from C to B' )
            self.view.prompt = 'Copying from C to B'
            item = self.view.current_item()
            self.model.request_operation( "copy_c2b", item )
        else:
            self.view.prompt = 'Invalid choice'
        self.input_handler = self.three_way_normal_input


    def three_way_normal_input( self, key_value ):
        ##print( key_value )
        self.view.prompt = ''

        if key_value == curses.KEY_UP:
            self.view.cursor_up()
        elif key_value == curses.KEY_DOWN:
            self.view.cursor_down()
        elif key_value == curses.KEY_PPAGE:
            self.view.scroll( -(self.canvas.rows - 2) )
        elif key_value == curses.KEY_NPAGE:
            self.view.scroll( self.canvas.rows - 2 )
        elif key_value == curses.KEY_LEFT or key_value == curses.KEY_RIGHT:
            self.view.toggle_collapse()
        elif key_value == 10 or key_value == curses.KEY_ENTER:
            current_item = self.view.current_item()
            if current_item.way() == 2:
                self.file_diff_current_item()
            else:
                self.file_merge_current_item()
        elif key_value == 3: # Cntrl-C
            #print( 'key=', key_value )
            self.need_to_quit = True
        elif key_value == ord('a'):
            self.input_handler = self.copy_a_3way_input
            self.view.prompt = 'Copy from A (left) to:'
        elif key_value == ord('b'):
            self.input_handler = self.copy_b_3way_input
            self.view.prompt = 'Copy from B (right) to:'
        elif key_value == ord('c'):
            self.input_handler = self.copy_c_3way_input
            self.view.prompt = 'Copy from C (middle) to:'
        elif key_value == ord('d'):
            self.delete_current_item()
        elif key_value == ord('h'):
            self.toggle_hidden_current_item()
        elif key_value == ord('H'):
            self.toggle_render_hidden()
        elif key_value == ord('m'):
            self.merge_to_center_item()
        elif key_value == ord('M'):
            self.merge_to_center_selection()
        elif key_value == ord('n'):
            self.model.merge_all()
        elif key_value == ord('r'):
            self.refresh_current_item()
        elif key_value == ord('R'):
            current_item = self.view.current_item()
            current_item.set_resolution_status_of_tree( 'r' )
        elif key_value == ord('s'):
            self.toggle_select_current_item()
        elif key_value == ord('S'):
            self.input_handler = self.select_column_input
            self.view.prompt = 'Select from which column (A, B, C):'
        elif key_value == ord('q'):
            #print( 'key=', key_value )
            self.need_to_quit = True


    def two_way_normal_input( self, key_value ):
        ##print( key_value )

        if key_value == curses.KEY_UP:
            self.view.cursor_up()
        elif key_value == curses.KEY_DOWN:
            self.view.cursor_down()
        elif key_value == curses.KEY_PPAGE:
            self.view.scroll( -(self.canvas.rows - 2) )
        elif key_value == curses.KEY_NPAGE:
            self.view.scroll( self.canvas.rows - 2 )
        elif key_value == curses.KEY_LEFT or key_value == curses.KEY_RIGHT:
            self.view.toggle_collapse()
        elif key_value == 10 or key_value == curses.KEY_ENTER:
            current_item = self.view.current_item()
            if current_item.way() == 2:
                self.file_diff_current_item()
            else:
                self.file_merge_current_item()
        elif key_value == 3: # Cntrl-C
            #print( 'key=', key_value )
            self.need_to_quit = True
        elif key_value == ord('a'):
            self.copy_current_item_l2r()
        elif key_value == ord('b'):
            self.copy_current_item_r2l()
        elif key_value == ord('d'):
            self.delete_current_item()
        elif key_value == ord('h'):
            self.toggle_hidden_current_item()
        elif key_value == ord('H'):
            self.toggle_render_hidden()
        elif key_value == ord('r'):
            self.refresh_current_item()
        elif key_value == ord('s'):
            self.toggle_select_current_item()
        elif key_value == ord('q'):
            #print( 'key=', key_value )
            self.need_to_quit = True


    def copy_current_item_l2r( self ):
        item = self.view.current_item()
        self.model.request_operation( "copy_l2r", item )


    def copy_current_item_r2l( self ):
        item = self.view.current_item()
        self.model.request_operation( "copy_r2l", item )


    def merge_to_center_item( self ):
        item = self.view.current_item()
        self.model.request_operation( "merge_to_center", item )

    def merge_to_center_selection( self ):
        self.model.request_operation( "merge_to_center", None )


    def file_merge_current_item( self ):
        current_item = self.view.current_item()

        if current_item.count() == 3:
            self.canvas.pre_external_command()
            self.filemerge.merge( current_item.left,
                                  current_item.middle,
                                  current_item.right )
            self.canvas.post_external_command()
        elif current_item.count() == 2:
            leftmost = current_item.left
            if leftmost is None:
                leftmost = current_item.middle

            rightmost = current_item.right
            if rightmost is None:
                rightmost = current_item.middle

            self.canvas.pre_external_command()
            self.filemerge.diff( leftmost, rightmost )
            self.canvas.post_external_command()
            self.model.request_operation( "refresh", current_item )
        elif current_item.count() == 1:
            if current_item.left is not None:
                path = current_item.left
            elif current_item.middle is not None:
                path = current_item.middle
            else:
                path = current_item.right

            if os.path.isdir( path ):
                # fixme: flash terminal
                pass
            else:
                self.canvas.pre_external_command()
                self.filemerge.view( path )
                self.canvas.post_external_command()
        else:
            # fixme: flash terminal
            pass



    def file_diff_current_item( self ):
        current_item = self.view.current_item()
        #print( "Left:", current_item.left )
        #print( "Right:", current_item.right )

        # We probably want to allow mixing and matching file viewers
        # and diff/merge programs, instead of having them combined in
        # one FileMerge object.
        if current_item.left is None:
            if os.path.isdir( current_item.right ):
                # fixme: flash terminal
                pass
            else:
                self.canvas.pre_external_command()
                self.filemerge.view( current_item.right )
                self.canvas.post_external_command()
        elif current_item.right is None:
            if os.path.isdir( current_item.left ):
                # fixme: flash terminal
                pass
            else:
                self.canvas.pre_external_command()
                self.filemerge.view( current_item.left )
                self.canvas.post_external_command()
        else:
            self.canvas.pre_external_command()
            self.filemerge.diff( current_item.left, current_item.right )
            self.canvas.post_external_command()
            self.model.request_operation( "refresh", current_item )


    def delete_current_item( self ):
        selection_exists = self.model.selection_exists()

        if selection_exists:
            item = None
        else:
            item = self.view.current_item()
        self.print_path_list( self.view.current )

        # fixme: the following request is should be synchronous, since
        # we need to know whether it succeeded to know whether or not
        # to reset the cursor.
        need_to_reset_cursor = self.model.request_operation( "delete", item )
        #print( 'need_to_reset_cursor=', need_to_reset_cursor )
        if need_to_reset_cursor:
            self.view.reset_cursor_after_delete()


    def print_path_list( self, path_list ):
        i = 0
        for item in path_list:
            #print( "item[", i , "]=", item.left )
            i += 1


    def refresh_current_item( self ):
        item = self.view.current_item()
        self.print_path_list( self.view.current )
        #print( "Refreshing: ", item.branch() )
        self.model.request_operation( "refresh", item )
        #item.refresh()

    def toggle_select_current_item( self ):
        item = self.view.current_item()
        item.toggle_selected()

    def toggle_hidden_current_item( self ):
        item = self.view.current_item()
        item.toggle_hidden()

    def toggle_render_hidden( self ):
        self.view.render_hidden = not self.view.render_hidden
