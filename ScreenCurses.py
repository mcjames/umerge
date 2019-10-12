# ScreenCurses.py

# This class represents the ncurses drawing surface. All ncurses calls
# will be encapsulated in this class.

import curses
import os
import time
import fcntl
import termios
import signal
import struct


# Colors for 16 color terminals: 0:black, 1:red, 2:green, 3:yellow,
# 4:blue, 5:magenta, 6:cyan, and 7:white.  These are the only
# available ones for backgrounds.  For foregrounds, you can also add
# "bright" to each of these.

class Canvas:

    def __init__( self ):
        #print( 'Creating canvas...' )
        self.rows = 0
        self.cols = 0
        self.need_to_resize = True
        self.need_full_refresh = False

        self.front = [[]]
        self.back = [[]]


    def destroy( self ):
        #print( 'Destroying the canvas...' )
        pass


    def init_curses( self ):
        # Does the terminal support color?
        self.has_colors = curses.has_colors()
        if self.has_colors:
            #print( 'Terminal supports color.' )
            curses.use_default_colors()
        else:
            #print( 'Terminal does not support color.' )
            pass

        # How many colors does it support?
        #print( 'Max colors:', self.max_colors() )

        # How many color pairs does it support?
        self.max_color_pairs = curses.COLOR_PAIRS
        #print( 'Max color pairs:', self.max_color_pairs )

        self.cols, self.rows = self.get_cols_rows()


    def init_color_pair( self, index, fg, bg ):
        if self.has_colors:
            curses.init_pair( index, fg, bg )


    def pre_external_command( self ):
        curses.endwin()


    def post_external_command( self ):
        self.stdscr.keypad( 1 )
        self.stdscr.clearok( 1 )
        self.set_full_refresh()
        self.need_to_resize = True

    def wrapper( self, func, *args, **kwds ):
        try:
            #print( 'in wrapper' )
            self.stdscr = curses.initscr()
            #print( '1' )
            curses.noecho()
            #print( '2' )
            curses.cbreak()
            #print( '3' )
            self.stdscr.keypad( 1 )
            #print( '4' )
            self.original_curs_set = curses.curs_set( 0 )
            #print( '5' )

            # Start color, too.  Harmless if the terminal doesn't have
            # color; user can test with has_color() later on.  The try/catch
            # works around a minor bit of over-conscientiousness in the curses
            # module -- the error return from C start_color() is ignorable.
            try:
                curses.start_color()
                #print( '6' )
            except:
                pass

            #print( '7' )
            self.init_curses()
            #print( '8' )
            self.resize()
            #print( '9' )
            self.install_sigwinch_handler();
            #print( '10' )

            return func( *args, **kwds )
        finally:
            # Set everything back to normal
            self.remove_sigwinch_handler();
            curses.curs_set( self.original_curs_set )
            self.stdscr.keypad( 0 )
            curses.echo()
            curses.nocbreak()
            #self.stdscr.clear()
            #self.stdscr.refresh()
            curses.endwin()

        #print( "Returning from wrapper" )


    def install_sigwinch_handler( self ):
        signal.signal( signal.SIGWINCH, self.sigwinch_handler )


    def remove_sigwinch_handler( self ):
        signal.signal( signal.SIGWINCH, signal.SIG_DFL )


    def sigwinch_handler( self, signo, frame ):
        self.need_to_resize = True


    def resize( self ):
        # SIGWINCH signals may come in very rapidly on some platforms
        # (e.g. on OS X, where it dynamically resizes as the user
        # drags the window border), so need_to_resize may get set
        # while we're handling the previous lines.  Keep looping here
        # until we're done receiving signals.

        #print( 'In resize() of canvas' )
        while True:
            self.need_to_resize = False
            self.cols, self.rows = self.get_cols_rows()
            if self.need_to_resize == False:
                break

        curses.resizeterm( self.rows, self.cols )
        self.front = [[None] * self.cols for i in range(self.rows)]
        self.back = [[None] * self.cols for i in range(self.rows)]
        self.clear()

    # addstr() returns an error sometimes when resizing the canvas.  I
    # should figure out how to prevent it, but I should be catching
    # the error anyway.
    def draw_text( self, x, y, text, color ):
        #print( "...x=", x )
        #print( "y=", x )
        #print( "text=", text )
        #print( "color=", color )

# old
#         if color == NORMAL_H or color == BLUE_H or color == GREEN_H:
#             self.stdscr.addstr( x, y, text,
#                                 curses.color_pair(color) | curses.A_BOLD )
#         else:
#             self.stdscr.addstr( x, y, text, curses.color_pair(color) )

# new
#        self.stdscr.addstr( x, y, text,
#                            curses.color_pair(color) )# | curses.A_BOLD )

        # Hack to work around the View sending us text that is too
        # long. fixme: remove this later.
        #print( 'canvas: length of row 0=', len(self.back[0]) )
        #amount_over = len(text) - 120
        #if amount_over > 0:
        #    text = text[:-amount_over]

        for i in range( len(text) ):
#             print( '\ntext=', text )
#             print( 'len(text)=', len(text) )
#             print( 'i=', i, ',text[i]=', text[i] )
#             print( 'x=', x )
#             print( 'y=', y )
#             print( 'len(back)=', len(self.back) )
#             print( 'len(back[0])=', len(self.back[0]) )
            self.back[x][y+i] = (text[i], color)


    def set_view( self, view ):
        self.view = view


    def getch( self ):
        return self.stdscr.getch()


    def get_input( self, timeout ):
        # If timeout is zero, block indefinitely.
        # Else if timeout > 0, that value is the number of 0.1 s to wait.

        # Wait for input with a timeout. If we timeout and no input has
        # occured, do nothing and return.
        if timeout != 0:
            #print( 'mike' )
            try:
                curses.halfdelay( timeout ) # tenths of a second
                key_value = self.getch()
            except:
                key_value = 3 # Control-C
            finally:
                curses.nocbreak() # Need both of these or just one?
                curses.cbreak()
        else:
            key_value = self.getch()

        return key_value


    # clear() and refresh() may need to change their names once it
    # becomes more clear how to better organize this stuff.

    def clear( self ):
        #print( 'clearing...' )

        if self.need_full_refresh:
            self.need_full_refresh = False
            #print( 'full_refresh clearing...' )
            # fixme: set background color to what user requests
            self.stdscr.clear()
            self.front = [[('', 0)] * self.cols for i in range(self.rows)]
        else:
            self.front = self.back

        self.back = [[('', 0)] * self.cols for i in range(self.rows)]


        # clear() needs to switch front and back, and write Nones to
        # all of the items in back.
#         self.front, self.back = self.back, self.front
#         for col in range( len( self.back[0] )):
#             for row in range( len( self.back )):
#                 self.back[row][col] = ('', 0)

#        self.front = self.back
#        self.back = [[('', 0)] * self.cols for i in range(self.rows)]



    def max_colors( self ):
        return curses.COLORS


    def set_full_refresh( self ):
        self.need_full_refresh = True
        self.stdscr.clear()
        #print( 'full_refresh clearing...' )
        self.front = [[('', 0)] * self.cols for i in range(self.rows)]


    def refresh( self ):
        # refresh() needs to look at back and then call addstr() for
        # all of the contents.

        # fixme: I could probably make this a generator or something
        # more Pythonic.
        segment = self.find_next_segment( 0, 0 )
        #print( 'found_segment=', segment )
        while segment:
            next = self.render_segment( segment[0], segment[1] )
            #print( 'next_pos=', next )
            segment = self.find_next_segment( next[0], next[1] )
            #print( 'found_segment=', segment )

        self.stdscr.refresh()



    # Returns a tuple of the row and col of the start of the next
    # segment to render.
    def find_next_segment( self, row, col ):
        while row < len( self.back ):
            while col < len( self.back[0] ):
                if self.front_and_back_differ( row, col ):
                    return (row, col)
                else:
                    col += 1
            row += 1
            col = 0

        return None


    # Returns (row,col) of the character *after* the last one rendered.
    def render_segment( self, row, col ):
        source = self.back[row]

        max_col = len( source )
        text = source[col][0]
        current_color = source[col][1]
        start_col = col

        col += 1
        while col < max_col:
            if ( source[col][1] == current_color
                 #and col != max_col - 1
                 and self.front_and_back_differ( row, col ) ):
                text += source[col][0]
                col += 1
            else:
                break

        if self.has_colors:
            self.stdscr.addstr( row, start_col, text,
                                curses.color_pair(current_color) )
                                # | curses.A_BOLD )
        else:
            self.stdscr.addstr( row, start_col, text )
                                # | curses.A_BOLD )

        next_row = row
        next_col = col
        if col == max_col:
            next_row += 1
            next_col = 0
        return ( next_row, next_col )


    def front_and_back_differ( self, row, col ):
        # fixme: get rid of the if statement
        if ( self.back[row][col][0] != self.front[row][col][0]
             or self.back[row][col][1] != self.front[row][col][1] ):
            return True
        else:
            return False



    # This is the only thing that I found that works.
    def get_cols_rows( self ):
        buffer = fcntl.ioctl( 0, termios.TIOCGWINSZ, '    ' )
        y, x = struct.unpack( 'hh', buffer )
        return x, y
