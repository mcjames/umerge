#!/usr/bin/env python3.1

# Match.py

import os
import threading
import time
import subprocess


UNENUMERATED = 0
UNCOMPARED = 1
SAME = 2
MISSING = 3
DIFFERENT = 4
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
        # print "  right: ", right
        self.parent = parent
        self.left = left
        self.right = right

        self.children = []
        self.num_diffs = 0
        self.collapse = False
        self.state = UNCOMPARED

    def way(self):
        return 2

    def mark_as_unenumerated(self):
        self.state = UNENUMERATED

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
        print("lrp:", self.top().left)
        return self.top().left

    def right_root_pathname(self):
        print("rrp:", self.top().right)
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

    def refresh(self):
        # If we are refreshing, we'll always need to remove the children.
        self.remove_children()

        # See if both left and right are still present (they might have
        # been deleted).
        left_name = self.left_pathname()
        right_name = self.right_pathname()

        left_present = os.path.exists(left_name)
        right_present = os.path.exists(right_name)

        # If they are both gone, so tell our parent to remove us
        # from the tree.
        if not left_present and not right_present:
            # fixme: what if we delete both roots? There's nothing left
            #    to compare.
            if self.parent:
                self.parent.remove_child_from_children(self)
            return

        # If at least one of them is still here, this node will stay in
        # place. The children are already removed, so just reenumerate().
        self.set_state_of_tree(UNCOMPARED)
        self.num_diffs = 0
        self.enumerate()
        # At this point, we need to do the comparison, but that needs
        # to be instigated from a higher level. It think that it's
        # probably time to redesign the code so that these things
        # can happen in the right places.

    def remove_child_from_children(self, child_to_remove):
        self.children.remove(child_to_remove)

    def remove_children(self):
        for child in self.children:
            child.remove_children()
        self.parent = None

    def enumerate(self):

        self.children = []

        # If both are None, don't do anything. Can this ever happen?
        if self.left is None and self.right is None:
            return

        if (self.left is not None and not os.path.isdir(self.left)
                and self.right is not None and not os.path.isdir(self.right)):
            return

        if (self.left is None and not os.path.isdir(self.right)
                or self.right is None and not os.path.isdir(self.left)):
            return

        leftfiles = []
        if self.left is not None and os.path.isdir(self.left):
            # print "\nLeft files:"
            leftfiles = os.listdir(self.left)
            leftfiles.sort(key=str.lower)
            # for file in leftfiles:
            #    print file

        rightfiles = []
        if self.right is not None and os.path.isdir(self.right):
            # print "\nRight files:"
            rightfiles = os.listdir(self.right)
            rightfiles.sort(key=str.lower)
            # for file in rightfiles:
            #    print file

        # Now merge the two lists together
        while (len(leftfiles) > 0 or len(rightfiles) > 0):
            if (len(rightfiles) == 0
                    or (len(leftfiles) > 0
                        and leftfiles[0].lower() < rightfiles[0].lower())):
                name = os.path.join(self.left, leftfiles[0])
                leftfiles = leftfiles[1:]
                self.children.append(Match(name, None, self))
            elif (len(leftfiles) == 0
                    or (len(rightfiles) > 0
                        and rightfiles[0].lower() < leftfiles[0].lower())):
                name = os.path.join(self.right, rightfiles[0])
                rightfiles = rightfiles[1:]
                self.children.append(Match(None, name, self))
            else:  # leftfiles[0] == rightfiles[0]
                assert(leftfiles[0].lower() == rightfiles[0].lower())
                leftname = os.path.join(self.left, leftfiles[0])
                rightname = os.path.join(self.right, rightfiles[0])
                leftfiles = leftfiles[1:]
                rightfiles = rightfiles[1:]
                self.children.append(Match(leftname, rightname, self))

        # Enumerate all the children
        for child in self.children:
            child.enumerate()

    def has_uncompared_descendants(self):
        if self.state == UNCOMPARED:
            return True
        for child in self.children:
            if child.has_uncompared_descendants():
                return True
        return False

    def compare(self):
        if self.state == UNCOMPARED:
            self.compare_myself()
            time.sleep(0.001)
            for child in self.children:
                child.compare()

    def compare_myself(self):
        if self.left is None or self.right is None:
            self.state = MISSING
        elif os.path.isdir(self.left) and os.path.isdir(self.right):
            self.state = SAME
        elif os.path.isdir(self.left) != os.path.isdir(self.right):
            # self.state = DIFFERENT
            self.set_state_of_tree(DIFFERENT)
        else:
            # command1 = "cmp -s " + self.left + " " + self.right
            # If the sizes differ, we avoid a cmp.
            if (os.path.getsize(self.left) == os.path.getsize(self.right)):
                # and os.system(command1) == 0):
                self.state = SAME
            else:
                p1 = subprocess.Popen(["diff", self.left, self.right],
                                      stdout=subprocess.PIPE)
                p2 = subprocess.Popen(["grep", "^[0-9]"],
                                      stdin=p1.stdout, stdout=subprocess.PIPE)
                p3 = subprocess.Popen(["wc", "-l"], stdin=p2.stdout,
                                      stdout=subprocess.PIPE)
                p1.wait()
                p2.wait()
                p3.wait()

                if p3.returncode == 0:
                    output = p3.communicate()[0]
                    self.num_diffs = int(output)
                    if self.num_diffs == 0:  # need this?
                        self.state = SAME
                    else:
                        self.state = DIFFERENT
                else:
                    print('the popen thing was terminated')

# Might be want some sort of optimization like this? modtimes are not very
# useful, but
#     if os.path.getsize(self.left) == os.path.getsize(self.right):
#         if os.path.getmtime(self.left)==os.path.getmtime(self.right)):
#             self.state = SAME

    def set_state_of_tree(self, new_state):
        # print("Setting state for: ", self.left)
        self.state = new_state
        for child in self.children:
            child.set_state_of_tree(new_state)

    def FilesAreUncompared(self):
        return self.state == UNCOMPARED

    def FilesAreSame(self):
        return self.state == SAME

    def FilesAreDifferent(self):
        return self.state == DIFFERENT

    def find_following_child(self, item):
        count = len(self.children)

        i = 0
        while i < count:
            if self.children[i] == item:
                if i == count - 1:
                    return None
                else:
                    return self.children[i+1]
            i += 1

        # If we get here, it's because item was not one of the
        # children. This should never happen.
        assert(False)

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
