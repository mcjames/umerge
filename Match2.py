# Match2.py

import os
import threading
import time
import subprocess
import sys

# Actually, am I still even using some of these states? It looks like
# I switched over to a different way of doing things at some point,
# and these are vestigial.
UNENUMERATED = 0
UNCOMPARED = 1
SAME = 2
MISSING = 3  # Unused?
DIFFERENT = 4  # Unused?
ERROR = 5
DELETED = 6


class Match:

    # It's not clear to me that we need this. Reads and writes of
    # integer variables are atomic in Python, so once the enumeration
    # is finished, the structure of the tree is fixed, and the only
    # thing happening during comparison is reading and writing of
    # integer variables.
    #
    # We would need the lock if we wanted to render during
    # enumeration, but my design doesn't for that atm. If we do
    # enumeration first and then comparison, the enumeration happens
    # so quickly that there's no point in adding the complexity to
    # render while enumerating.
    # lock = threading.Lock()

    # left and right should be absolute pathnames.
    def __init__(self, left, right, parent):
        # print "Creating match with:"
        # print "  left : ", left
        # print "  middle : ", middle
        # print "  right: ", right
        self.parent = parent
        self.left = left
        self.right = right

        self.children = []
        self.num_diffs = None
        self.collapse = False
        self.state = UNENUMERATED
        self.selected = False
        self.hidden = False
        self.resolution_status = ' '

    # fixme: do not allow this method to run if the whole tree has not
    # finished comparing.
    def select(self, column, feature):
        self.selected = self.selection_matches(column, feature)
        for child in self.children:
            child.select(column, feature)

    def selection_matches(self, column, feature):
        # if self.left is not None and self.middle is None:
        if self.left is not None:  # 2FIX
            return True

        return False

    def unselect_all(self):
        self.selected = False
        for child in self.children:
            child.unselect_all()

    def selection_exists(self):
        if self.selected:
            return True
        for child in self.children:
            if child.selection_exists():
                return True
        return False

    def toggle_selected(self):
        self.set_selected(not self.selected)

    def set_selected(self, value):
        self.selected = value
        for child in self.children:
            child.set_selected(value)

    def toggle_hidden(self):
        self.set_hidden(not self.hidden)

    def set_hidden(self, value):
        self.hidden = value
        for child in self.children:
            child.set_hidden(value)

    def way(self):
        return 2

    def letter_to_subpart(self, letter):
        if letter == 'a':
            return self.left_pathname()
        else:
            # letter == 'b'
            return self.right_pathname()

    def toggle_collapse(self):
        self.collapse = not self.collapse

    def HasError(self):
        return self.state == ERROR

    def IsHidden(self):
        return self.state == DELETED

    def top(self):
        current = self
        ancestor = self.parent
        while ancestor is not None:
            current = ancestor
            ancestor = ancestor.parent
        return current

    def left_root_pathname(self):
        # print("lrp:", self.top().left)
        return self.top().left

    def right_root_pathname(self):
        # print("rrp:", self.top().right)
        return self.top().right

    def branch(self):
        if self.left:
            left_root = self.left_root_pathname()
            assert(self.left.startswith(left_root))
            return self.left[len(left_root)+1:]
        elif self.right:
            right_root = self.right_root_pathname()
            assert(self.right.startswith(right_root))
            return self.right[len(right_root)+1:]
        else:
            assert(False)

    def left_pathname(self):
        # print('in left_pathname()')
        if self.left:
            # print("lp: 1:", self.left)
            return self.left
        else:
            # print("lp: 2:")
            return os.path.join(self.left_root_pathname(), self.branch())

    def right_pathname(self):
        # print('in right_pathname()')
        if self.right:
            # print("rp: 1:", self.right)
            return self.right
        else:
            # print("rp: 2:")
            # print("    ", self.right_root_pathname())
            # print("    ", self.branch())
            # print("a   ", os.path.join(self.right_root_pathname(),
            #                             self.branch()))
            return os.path.join(self.right_root_pathname(), self.branch())

    def remove_child_from_children(self, child_to_remove):
        self.children.remove(child_to_remove)

    def remove_children(self):
        for child in self.children:
            child.remove_children()
        self.parent = None

    def __can_enumerate(self):
        # If both are None, don't do anything. Can this ever happen?
        if (self.left is None
                and self.right is None):
            return False

        # Is there at least one directory?
        left_is_dir = (self.left is not None
                       and os.path.isdir(self.left))
        right_is_dir = (self.right is not None
                        and os.path.isdir(self.right))

        if not left_is_dir and not right_is_dir:
            return False

#         if (self.left is not None and not os.path.isdir(self.left)
#             and self.right is not None and not os.path.isdir(self.right)):
#             return False

#         if (self.left is None and not os.path.isdir(self.right)
#             or self.right is None and not os.path.isdir(self.left)):
#             return False

        return True

    def enumerate(self):

        # print('Setting children to [] for:', self)
        self.children = []

        if not self.__can_enumerate():
            self.state = UNCOMPARED
            return

        leftfiles = []
        if self.left is not None and os.path.isdir(self.left):
            leftfiles = os.listdir(self.left)
            leftfiles.sort(key=str.lower)

        rightfiles = []
        if self.right is not None and os.path.isdir(self.right):
            rightfiles = os.listdir(self.right)
            rightfiles.sort(key=str.lower)

        # Now merge the three lists together
        while self.__enumeration_files_available(leftfiles,
                                                 rightfiles):

            lowest = self.__get_lowest(leftfiles, rightfiles)

            leftname = None
            if len(leftfiles) > 0:
                if leftfiles[0].lower() == lowest:
                    leftname = os.path.join(self.left, leftfiles[0])
                    leftfiles = leftfiles[1:]

            rightname = None
            if len(rightfiles) > 0:
                if rightfiles[0].lower() == lowest:
                    rightname = os.path.join(self.right, rightfiles[0])
                    rightfiles = rightfiles[1:]

            self.children.append(
                Match(leftname, rightname, self))

        # Enumerate all the children
        for child in self.children:
            child.enumerate()

        self.state = UNCOMPARED

    # Fixme: enumeration uses stuff all in lower case, but UNIX is case-
    # sensitive. I need to rewrite this and the 2-way version to handle
    # case correctly.
    def __get_lowest(self, leftfiles, rightfiles):
        candidates = []

        if len(leftfiles) > 0:
            candidates.append(leftfiles[0].lower())
        if len(rightfiles) > 0:
            candidates.append(rightfiles[0].lower())

        return min(candidates)

    def __enumeration_files_available(self, leftfiles,
                                      rightfiles):
        return (len(leftfiles) > 0
                or len(rightfiles) > 0)

    def has_uncompared_descendants(self):
        if self.state == UNCOMPARED:
            return True
        for child in self.children:
            if child.has_uncompared_descendants():
                return True
        return False

    # Am I still using this? It looks like this was how I did it
    # before I allowed the ability to interrupt the operation.
    # PyLint complains that compare_myself() takes an argument, which I'm not
    # passing in, so this may be dead code.  Get rid of it in the future.
    # def compare(self):
    #     if self.state == UNCOMPARED:
    #         self.compare_myself()
    #         time.sleep(0.001)
    #         for child in self.children:
    #             child.compare()

    def count(self):
        count = 0

        if self.left is not None:
            count += 1
        if self.right is not None:
            count += 1

        return count

    def compare_myself(self, fileops):
        left_pathname = self.left_pathname()
        right_pathname = self.right_pathname()

        if os.path.exists(left_pathname):
            self.left = left_pathname
        if os.path.exists(right_pathname):
            self.right = right_pathname

        # print('in compare_myself()')
        count = self.count()
        # print('   count = ', count)

        assert count != 0

        if count == 1:
            self.state = SAME
            return
        else:
            # count == 2
            if not self.__can_enumerate():
                self.num_diffs = fileops.compare_two_files(self.left,
                                                           self.right)
            else:
                self.num_diffs = None

        # fixme:
        self.state = SAME

    def set_state_of_tree(self, new_state):
        # print("Setting state for: ", self.left)
        self.state = new_state
        for child in self.children:
            child.set_state_of_tree(new_state)

    def set_resolution_status_of_tree(self, new_resolution_status):
        self.resolution_status = new_resolution_status
        for child in self.children:
            child.set_resolution_status_of_tree(new_resolution_status)

    def FilesAreUncompared(self):
        return self.state == UNCOMPARED

    def FilesAreSame(self):
        return self.state == SAME

    def FilesAreDifferent(self):
        if self.left is not None and os.path.isdir(self.left):
            return False
        if self.right is not None and os.path.isdir(self.right):
            return False
        return self.num_diffs != 0

    def find_following_child(self, item):
        count = len(self.children)

        i = 0
        # print('looking for: ', item, item.left)
        while i < count:
            # print('checking:', self.children[i], self.children[i].left)
            if self.children[i] == item:
                if i == count - 1:
                    return None
                else:
                    return self.children[i+1]
            i += 1

        # If we get here, it's because item was not one of the
        # children. This should never happen.
        # print('Failed:', item.left)
        # print('  self:', self.left)
        # print('children:')
        self.print_children()
        assert(False)

    def print_children(self):
        pass
        # for child in self.children:
        #     print('   ', child.left)

    def find_previous_child(self, item):
        count = len(self.children)

        i = 0
        while i < count:
            if self.children[i] == item:
                if i == 0:
                    return None
                else:
                    return self.children[i-1]
            i += 1

        # If we get here, it's because item was not one of the
        # children. This should never happen.
        assert(False)

    # This was from the old way of doing things. Get rid of it once
    # I'm sure that I don't need it.

    # ###########################################################################
    # #
    # # Since we don't re-enumerate, this is essentially just walking the tree
    # # and recomparing each item.
    # #
    # def refresh(self):
    #     # We probably need to set some sort of state on the
    #     # tree (UNCOMPARED?)
    #     # and then request a comparision. How can we do this from the match?

    #     # If we are refreshing, we'll always need to remove the children.
    #     self.remove_children()

    #     # See if both left and right are still present (they might have
    #     # been deleted).
    #     left_name = self.left_pathname()
    #     right_name = self.right_pathname()

    #     left_present = os.path.exists(left_name)
    #     right_present = os.path.exists(right_name)

    #     # If they are both gone, so tell our parent to remove us
    #     # from the tree.
    #     if not left_present and not right_present:
    #         # fixme: what if we delete both roots? There's nothing left
    #         #    to compare.
    #         print("about to remove_child_from_children():")
    #         print(left_name, right_name)
    #         print('parent:', parent, parent.left)
    #         if self.parent:
    #             self.parent.remove_child_from_children(self)
    #         return

    #     # If at least one of them is still here, this node will stay in
    #     # place. The children are already removed, so just reenumerate().
    #     self.set_state_of_tree(UNCOMPARED)
    #     self.lm_num_diffs = None
    #     self.mr_num_diffs = None
    #     self.lr_num_diffs = None
    #     self.enumerate()
    #     # At this point, we need to do the comparison, but that needs
    #     # to be instigated from a higher level. It think that it's
    #     # probably time to redesign the code so that these things
    #     # can happen in the right places.
