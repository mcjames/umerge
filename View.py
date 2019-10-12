#!/usr/bin/env python3.1

# View.py

import Model
import os


# This class takes the data from the model and renders it to the
# canvas. It knows nothing about the libraries that get the actual
# drawing done. It keeps track of where in the data we should be
# grabbing data to draw and keeps track of things like the cursor. It
# has no control capability.


NORMAL =    1
NORMAL_H =  2
BLUE =      3
BLUE_H =    4
GREEN =     5
GREEN_H =   6
GRAY =      7
GRAY_H =    8
RED =       9
RED_H =    10
STATUS =   11


class View:

    def __init__( self, canvas, model, settings ):

        self.settings = settings

        self.canvas = canvas
        canvas.set_view( self )

        self.__recalculate_sizes()

        self.model = model
        self.cursor = 0
        self.last_displayed_item_row = self.cursor
        self.current = [] # current is at [0], top is at length-1
        #self.top = [model.top]
        self.top = None
        self.spinner = 1

        self.__init_a_color_pair( NORMAL, 'normal_fg', 'normal_bg' )
        self.__init_a_color_pair( NORMAL_H, 'normal_h_fg', 'normal_h_bg' )
        self.__init_a_color_pair( BLUE, 'differ_fg', 'differ_bg' )
        self.__init_a_color_pair( BLUE_H, 'differ_h_fg', 'differ_h_bg' )
        self.__init_a_color_pair( GREEN, 'only_one_fg', 'only_one_bg' )
        self.__init_a_color_pair( GREEN_H, 'only_one_h_fg', 'only_one_h_bg' )
        self.__init_a_color_pair( GRAY, 'uncompared_fg', 'uncompared_bg' )
        self.__init_a_color_pair( GRAY_H, 'uncompared_h_fg', 'uncompared_h_bg')
        self.__init_a_color_pair( RED, 'error_fg', 'error_bg' )
        self.__init_a_color_pair( RED_H, 'error_h_fg', 'error_h_bg')
        self.__init_a_color_pair( STATUS, 'uncompared_bg', 'uncompared_fg')


    def __init_a_color_pair( self, index, fg_name, bg_name ):
        fg_number = self.settings.get_value( fg_name )
        bg_number = self.settings.get_value( bg_name )
        self.canvas.init_color_pair( index, fg_number, bg_number )


    def destroy( self ):
        pass


    def current_item( self ):
        assert( len(self.current) > 0 )
        return self.current[0]


    def reset_cursor_after_delete( self ):
        if self.cursor == 0:
            self.__reset_top()
        else:
            if self.__next_display_line( self.current ) is None:
                self.cursor -= 1


    def __reset_top( self ):
        current = self.top

        # fixme: can len(current) ever be zero?
        while current is not None and current[0].IsHidden():
            current = self.__next_display_line_aux( current )
            #print( 'path=', current )

        if current is None:
            # scroll up one page
            self.scroll( -self.rows ) # fixme: should be # of items
            self.cursor = self.last_item_row # fixme: should be last
                                             # valid item
        else:
            # Effectively make the next undeleted item the top one
            self.top = current


    def toggle_collapse( self ):
        self.current_item().toggle_collapse()


    def __can_scroll_down( self ):
        # fixme: not entirely correct
        return self.last_displayed_item_row == self.last_item_row


    def __can_scroll_up( self ):
        return True


    def scroll( self, number_of_lines ):
        print( "\nself.last_displayed_item_row:",
               self.last_displayed_item_row )
        print( "self.rows:", self.rows )
        print( "last_item_on_screen:", self.last_displayed_item_row )

        if number_of_lines > 0:
            if not self.__can_scroll_down():
                return

            self.canvas.set_full_refresh()
            i = 0
            while i < number_of_lines:
                # This works well to prevent crashes, but if you
                # page down and the last line is half-way up the
                # screen, it will make the last line "top." It might
                # be nice to execute the loop before setting top, and
                # not change top if we can't scroll a whole page. I'll
                # live with this awhile and see which I like better.
                next = self.__next_display_line( self.top )
                if next != None:
                    self.top = next
                i += 1
            if self.cursor > self.last_displayed_item_row:
                self.cursor = self.last_displayed_item_row
        else:
            if not self.__can_scroll_up():
                return

            self.canvas.set_full_refresh()
            number_of_lines = -number_of_lines
            i = 0
            while i < number_of_lines:
                prev = self.__prev_display_line( self.top )
                if prev != None:
                    self.top = prev
                i += 1


    def cursor_up( self ):
        if self.__prev_display_line( self.current ) is not None:
            if self.cursor > 0:
                self.cursor -= 1
            else:
                self.scroll( -1 )
        else:
            # fixme: Flash the console here.
            pass


    def cursor_down( self ):
        if self.__next_display_line( self.current ) is not None:
            if self.cursor < self.last_displayed_item_row:
                self.cursor += 1
            else:
                self.scroll( 1 )
        else:
            # fixme: Flash the console here.
            pass


    def render( self ):

        self.__render_items()
        self.__render_roots_line()
        self.__render_status_line()
        self.__render_command_line()

        self.canvas.refresh()


    def __render_roots_line( self ):
        # Render the top line here with the model.top directories
        # on it. In order to get this right, I'll have to clean up
        # things to compute the rest of the lines right if I start on
        # row 1 rather than row 0.
        pass

    def __render_status_line( self ):
        test_string = "          " * 15

        #print( 'view.cols=', self.cols )
        amount_over = len(test_string) - self.cols
        #print( 'amount_over=', amount_over )
        if amount_over > 0:
            test_string = test_string[:-amount_over]

        #print( 'final length=', len( test_string) )
        self.canvas.draw_text( self.status_row, 0, test_string, STATUS )

        spinner = ' |/-\\'
        if self.model.state() == Model.STATE_NORMAL:
            self.spinner = 0
        else:
            self.spinner += 1
            if self.spinner == 5: # len(spinner)
                self.spinner = 1
            self.canvas.draw_text( self.status_row, 50, spinner[self.spinner],
                                   STATUS )


    def __render_command_line( self ):
        # Curses cannot display in the last character of the last
        # line, so shorten it by amount_over + 1.
        test_string = "          " * 15
        #print( 'view.cols=', self.cols )
        amount_over = len(test_string) - self.cols
        #print( 'amount_over=', amount_over )
        if amount_over > 0:
            test_string = test_string[:-(amount_over+1)]

        #print( 'final length=', len( test_string) )
        self.canvas.draw_text( self.command_row, 0, test_string, NORMAL )

        if self.model.state() == Model.STATE_ENUMERATING:
            self.canvas.draw_text( self.command_row, 0, 'Enumerating...',
                                   NORMAL )
        else:
            self.canvas.draw_text( self.command_row, 0, '              ',
                                   NORMAL )


    def __render_items( self ):
        #print( '\nIn View.render()' )
        if self.model.state() == Model.STATE_ENUMERATING:
            self.canvas.clear()
        else:
            with self.model.lock():
                #print( 'start of render' )
                self.canvas.clear()

                if self.top is None:
                    self.top = [self.model.top.children[0], self.model.top]

                row = 0
                current = self.top

                #print( 'rows=', self.rows )
                #print( 'cols=', self.cols )
                while row <= self.last_item_row:
                    if row == self.cursor:
                        self.current = current
                    self.__render_path_item( row, current )
                    self.last_displayed_item_row = row
                    row += 1
                    current = self.__next_display_line( current )
                    if current == None:
                        break

                # fixme: Not quite right. Cursor gets set properly, but
                # self.current doesn't get updated and we need to re-render.
                #if self.cursor > self.last_displayed_item_row:
                #    self.cursor = self.last_displayed_item_row
                while row <= self.last_item_row:
                    self.__render_empty_item( row )
                    row += 1



    def resize( self ):
        print( '\nResizing...' )
        self.canvas.resize()
        self.__recalculate_sizes()
        self.canvas.set_full_refresh()


    def __recalculate_sizes( self ):
        # Now recompute the locations of various things.
        self.rows = self.canvas.rows
        self.cols = self.canvas.cols

        print( '  new rows=', self.rows )
        print( '  new cols=', self.cols )

        # fixme: If there are less than four rows, don't render and just
        # print an error that the display is too small.
        self.roots_row = 0
        self.first_item_row = 0 # fixme: make this = 1
        self.last_item_row = self.rows - 3
        self.status_row = self.rows - 2
        self.command_row = self.rows - 1

        # We use self.cols - 1 since there is a vertical bar in the middle.
        self.left_item_width = (self.cols - 1) // 2
        self.right_item_width = self.left_item_width
        if (self.cols - 1) % 2:
            self.left_item_width += 1
        assert( self.left_item_width + self.right_item_width + 1 == self.cols )
        print( 'left_item_width =', self.left_item_width )
        print( 'right_item_width=', self.right_item_width )


    def __render_path_item( self, row, path_item ):
        indention = 4 * ( len(path_item) - 2 )
        item = path_item[0]

        #print( 'left width =', self.left_item_width )
        #print( 'right width=', self.right_item_width )
        if item.left == None:
            leftside = " " * self.left_item_width
        else:
            leftside = " " * indention

            if os.path.isdir( item.left ):
                if item.collapse:
                    leftside += '\u25B6 ' #'- '
                else:
                    leftside += '\u25BC ' #'o '
            else:
                leftside += '  '

            leftside += os.path.basename( item.left )
            #if not os.path.isdir( item.left ):
            if item.num_diffs > 0:
                leftside += "---" + str(item.num_diffs)
            if len(leftside) < self.left_item_width:
                leftside += " " * ( self.left_item_width - len(leftside) )
            elif len(leftside) > self.left_item_width:
                leftside = leftside[:-(len(leftside)-self.left_item_width)]

        if item.right == None:
            rightside = " " * self.right_item_width
        else:
            rightside = " " * indention

            if os.path.isdir( item.right ):
                if item.collapse:
                    rightside += '\u25B6 ' #'- '
                else:
                    rightside += '\u25BC ' #'o '
            else:
                rightside += '  '

            rightside += os.path.basename( item.right )
            #if not os.path.isdir( item.right ):
            if item.num_diffs > 0:
                rightside += "---" + str(item.num_diffs)
            if len(rightside) < self.right_item_width:
                rightside += " " * ( self.right_item_width - len(rightside) )
            elif len(rightside) > self.right_item_width:
                rightside = rightside[:-(len(rightside)-self.right_item_width)]

        finalline = leftside + "|" + rightside

        # fixme: It looks like current_item is getting set while
        # we render. That's a lousy design. It should be set when we
        # are actually moving the cursor.
        #if row == self.cursor:
        #    self.current_item = item

        if item.HasError():
            self.__print_red( row, finalline )
        elif item.FilesAreDifferent():
            self.__print_blue( row, finalline )
        elif item.left == None or item.right == None:
            self.__print_green( row, leftside, rightside )
        elif item.FilesAreUncompared():
            self.__print_gray( row, finalline )
        else:
            #print finalline
            self.__render_line( row, 0, finalline, NORMAL )


    def __print_gray( self, row, finalline ):
        self.__render_line( row, 0, finalline, GRAY )

    def __print_red( self, row, finalline ):
        self.__render_line( row, 0, finalline, RED )

    def __print_blue( self, row, finalline ):
        self.__render_line( row, 0, finalline, BLUE )

    def __print_green( self, row, leftside, rightside ):
        if leftside == (" " * self.left_item_width):
            self.__render_line( row, 0,
                                leftside + "|", NORMAL )
            self.__render_line( row, len(leftside) + 1,
                                rightside, GREEN )
        else:
            self.__render_line( row, 0,
                                leftside, GREEN )
            self.__render_line( row, len(leftside),
                                "|" + rightside, NORMAL )


    def __render_line( self, row, col, text, color ):
        if row == self.cursor:
            if color == NORMAL:
                color = NORMAL_H
            elif color == BLUE:
                color = BLUE_H
            elif color == GREEN:
                color = GREEN_H
            elif color == GRAY:
                color = GRAY_H
            elif color == RED:
                color = RED_H
        self.canvas.draw_text( row, col, text, color )


    def __render_empty_item( self, row ):
        out_string = ( ' ' * self.left_item_width + '|'
                       + ' ' * self.right_item_width )
        self.__render_line( row, 0, out_string, NORMAL )


    def __next_display_line( self, path_const ):
        candidate_path = self.__next_display_line_aux( path_const )
        if candidate_path is None:
            #print( 'None' )
            return None
        #print( 'path=', candidate_path[0].left )

        while candidate_path[0].IsHidden():
            candidate_path = self.__next_display_line_aux( candidate_path )
            #print( 'path=', candidate_path )
            if candidate_path is None:
                return None

        return candidate_path


    def __next_display_line_aux( self, path_const ):
        path = path_const[:]
        current = path[0]

        if not current.collapse:
            # If the current item is not a leaf node, then the first child
            # is "next".
            if len( current.children ) > 0:
                path.insert( 0, current.children[0] )
                return path

        # Otherwise, we're a leaf node, so the next is the sibling
        # that follows us in our parent's list of children. If we are
        # our parent's last sibling, we try the the node that follows
        # our parent in the grandparent's list, and so on. If at any
        # time we try to move up the tree and the ancestor is None (we
        # are at the last item in path[]), return None, since we must
        # be the last item in the tree.
        if len(path) == 1:
            # We're the root node, so there's no next.
            return None
        parent = path[1]

        while True:
            next = parent.find_following_child( current )
            if next != None:
                path[0] = next
                return path
            else:
                path.pop( 0 )
                if len(path) == 1:
                    return None
                current = path[0]
                parent = path[1]


    def __prev_display_line( self, path_const ):
        candidate_path = self.__prev_display_line_aux( path_const )
        if candidate_path is None:
            return None

        while candidate_path[0].IsHidden():
            candidate_path = self.__prev_display_line_aux( candidate_path )
            if candidate_path is None:
                return None

        return candidate_path


    def __prev_display_line_aux( self, path_const ):
        # If we are at the root node, just return None
        #if len(path_const) == 1:
        #    return None

        if ( len(path_const) == 2
             and path_const[0] == self.model.top.children[0] ):
            return None

        path = path_const[:]
        current = path[0]
        parent = path[1]

        prev = parent.find_previous_child( current )
        if prev == None:
            path.pop( 0 )
        else:
            current = prev
            path[0] = prev

            while True:
                if current.collapse:
                    break
                number_children = len( current.children )
                if number_children > 0:
                    current = current.children[number_children - 1]
                    path.insert( 0, current )
                else:
                    break

        return path
